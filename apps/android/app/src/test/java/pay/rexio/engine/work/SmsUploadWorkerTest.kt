package pay.rexio.engine.work

import android.content.Context
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.work.ListenableWorker
import androidx.work.WorkerFactory
import androidx.work.WorkerParameters
import androidx.work.testing.TestListenableWorkerBuilder
import com.google.common.truth.Truth.assertThat
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.OkHttpClient
import org.junit.After
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import pay.rexio.engine.data.local.OutboxRepository
import pay.rexio.engine.data.local.RexioDatabase
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.remote.DeviceApiProvider

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class SmsUploadWorkerTest {
    private lateinit var server: MockWebServer
    private lateinit var db: RexioDatabase
    private lateinit var outbox: OutboxRepository
    private lateinit var session: SessionStore
    private lateinit var prefs: PrefsStore

    private val context: Context = ApplicationProvider.getApplicationContext()
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        encodeDefaults = false
    }

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        db = Room.inMemoryDatabaseBuilder(context, RexioDatabase::class.java)
            .allowMainThreadQueries()
            .build()
        outbox = OutboxRepository(db.outboxDao())
        session = SessionStore(context)
        prefs = PrefsStore(context, "upload_test_${System.nanoTime()}")
        runBlocking { prefs.setServerUrl(server.url("/").toString()) }
    }

    @After
    fun tearDown() {
        server.shutdown()
        db.close()
    }

    private fun worker(apiProvider: DeviceApiProvider = defaultApiProvider()): SmsUploadWorker {
        val factory = object : WorkerFactory() {
            override fun createWorker(
                appContext: Context,
                workerClassName: String,
                workerParameters: WorkerParameters,
            ): ListenableWorker = SmsUploadWorker(
                context = appContext,
                params = workerParameters,
                outboxRepository = outbox,
                sessionStore = session,
                prefsStore = prefs,
                apiProvider = apiProvider,
                json = json,
            )
        }
        return TestListenableWorkerBuilder<SmsUploadWorker>(context)
            .setWorkerFactory(factory)
            .build()
    }

    private fun defaultApiProvider() = DeviceApiProvider(
        OkHttpClient.Builder().connectTimeout(5, TimeUnit.SECONDS).build(),
        json,
    )

    @Test
    fun `uploads batch and deletes acked rows`() = runTest {
        session.setPaired("dev_abc", "mer_1")
        outbox.enqueue("bKash", "Tk 100", 0, 1_000L, "bkash")
        outbox.enqueue("NAGAD", "Tk 200", 1, 2_000L, "nagad")
        server.enqueue(
            MockResponse().setResponseCode(200).setBody(
                """{"acks":[{"client_id":"1","status":"parsed"},{"client_id":"2","status":"failed"}]}""",
            ),
        )

        val result = worker().doWork()

        assertThat(result).isInstanceOf(ListenableWorker.Result.Success::class.java)
        assertThat(outbox.pendingCountOnce()).isEqualTo(0)
        val recorded = server.takeRequest()
        assertThat(recorded.path).isEqualTo("/v1/device/sms")
        val body = recorded.body.readUtf8()
        assertThat(body).contains("\"client_id\":\"1\"")
        assertThat(body).contains("\"sim_slot\":0")
        assertThat(body).contains("\"provider\":\"nagad\"")
        assertThat(body).contains("\"sms_time\":\"1970-01-01T00:00:01Z\"")
    }

    @Test
    fun `server error retries and keeps the row queued`() = runTest {
        session.setPaired("dev_abc", null)
        outbox.enqueue("bKash", "Tk 100", null, 1_000L)
        server.enqueue(
            MockResponse().setResponseCode(500).setBody(
                """{"error":{"code":"internal_error","message":"boom"}}""",
            ),
        )

        val result = worker().doWork()

        assertThat(result).isInstanceOf(ListenableWorker.Result.Retry::class.java)
        assertThat(outbox.pendingCountOnce()).isEqualTo(1)
    }

    @Test
    fun `rate limit retries`() = runTest {
        session.setPaired("dev_abc", null)
        outbox.enqueue("bKash", "Tk 100", null, 1_000L)
        server.enqueue(
            MockResponse().setResponseCode(429).setBody(
                """{"error":{"code":"rate_limit_exceeded","message":"too many requests"}}""",
            ),
        )

        val result = worker().doWork()

        assertThat(result).isInstanceOf(ListenableWorker.Result.Retry::class.java)
    }

    @Test
    fun `io failure retries`() = runTest {
        session.setPaired("dev_abc", null)
        outbox.enqueue("bKash", "Tk 100", null, 1_000L)
        runBlocking { prefs.setServerUrl("http://127.0.0.1:1") }
        val deadApi = DeviceApiProvider(
            OkHttpClient.Builder().connectTimeout(2, TimeUnit.SECONDS).build(),
            json,
        )

        val result = worker(deadApi).doWork()

        assertThat(result).isInstanceOf(ListenableWorker.Result.Retry::class.java)
        assertThat(outbox.pendingCountOnce()).isEqualTo(1)
    }

    @Test
    fun `auth failure stops the chain and sets the signing cooldown`() = runTest {
        session.setPaired("dev_abc", null)
        outbox.enqueue("bKash", "Tk 100", null, 1_000L)
        server.enqueue(
            MockResponse().setResponseCode(401).setBody(
                """{"error":{"code":"device_disabled","message":"device is disabled"}}""",
            ),
        )

        val result = worker().doWork()

        assertThat(result).isInstanceOf(ListenableWorker.Result.Failure::class.java)
        val cooldownUntil = prefs.prefs.first().authCooldownUntil
        assertThat(cooldownUntil).isGreaterThan(System.currentTimeMillis() - 1_000)
        assertThat(outbox.pendingCountOnce()).isEqualTo(1)
    }

    @Test
    fun `unpaired worker succeeds without any request`() = runTest {
        val result = worker().doWork()
        assertThat(result).isInstanceOf(ListenableWorker.Result.Success::class.java)
        assertThat(server.requestCount).isEqualTo(0)
    }

    @Test
    fun `empty queue succeeds without any request`() = runTest {
        session.setPaired("dev_abc", null)
        val result = worker().doWork()
        assertThat(result).isInstanceOf(ListenableWorker.Result.Success::class.java)
        assertThat(server.requestCount).isEqualTo(0)
    }
}
