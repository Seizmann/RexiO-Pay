package pay.rexio.engine.domain.heartbeat

import android.content.Context
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import com.google.common.truth.Truth.assertThat
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
class HeartbeatEngineTest {
    private lateinit var server: MockWebServer
    private lateinit var db: RexioDatabase
    private lateinit var outbox: OutboxRepository
    private lateinit var session: SessionStore
    private lateinit var prefs: PrefsStore
    private lateinit var engine: HeartbeatEngine

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
        prefs = PrefsStore(context, "hb_test_${System.nanoTime()}")
        engine = HeartbeatEngine(
            apiProvider = DeviceApiProvider(OkHttpClient.Builder().build(), json),
            prefsStore = prefs,
            sessionStore = session,
            outboxRepository = outbox,
            battery = BatteryLevelProvider { 77 },
            json = json,
        )
        runBlocking { prefs.setServerUrl(server.url("/").toString()) }
    }

    @After
    fun tearDown() {
        server.shutdown()
        db.close()
    }

    @Test
    fun `sends battery app version and queue depth`() = runTest {
        session.setPaired("dev_abc", null)
        outbox.enqueue("bKash", "Tk 100", null, 1_000L)
        outbox.enqueue("NAGAD", "Tk 200", null, 2_000L)
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"status":"ok"}"""))

        val outcome = engine.heartbeatOnce()

        assertThat(outcome).isEqualTo(HeartbeatOutcome.Ok)
        val recorded = server.takeRequest()
        assertThat(recorded.path).isEqualTo("/v1/device/heartbeat")
        assertThat(recorded.body.readUtf8())
            .isEqualTo("""{"battery_level":77,"app_version":"1.0.0","queue_depth":2}""")
    }

    @Test
    fun `unpaired device skips the request`() = runTest {
        assertThat(engine.heartbeatOnce()).isEqualTo(HeartbeatOutcome.Unpaired)
        assertThat(server.requestCount).isEqualTo(0)
    }

    @Test
    fun `auth cooldown pauses signed traffic`() = runTest {
        session.setPaired("dev_abc", null)
        runBlocking { prefs.setAuthCooldownUntil(System.currentTimeMillis() + 60_000) }

        assertThat(engine.heartbeatOnce()).isEqualTo(HeartbeatOutcome.AuthPaused)
        assertThat(server.requestCount).isEqualTo(0)
    }

    @Test
    fun `a 401 engages the signing cooldown`() = runTest {
        session.setPaired("dev_abc", null)
        server.enqueue(
            MockResponse().setResponseCode(401).setBody(
                """{"error":{"code":"device_disabled","message":"device is disabled"}}""",
            ),
        )

        val outcome = engine.heartbeatOnce()

        assertThat(outcome).isInstanceOf(HeartbeatOutcome.Failed::class.java)
        assertThat(prefs.prefs.first().authCooldownUntil)
            .isGreaterThan(System.currentTimeMillis() - 1_000)
    }

    @Test
    fun `network failure reports offline without engaging cooldown`() = runTest {
        session.setPaired("dev_abc", null)
        runBlocking { prefs.setServerUrl("http://127.0.0.1:1") }

        val outcome = engine.heartbeatOnce()

        assertThat(outcome).isInstanceOf(HeartbeatOutcome.Failed::class.java)
        assertThat(prefs.prefs.first().authCooldownUntil).isEqualTo(0)
    }
}
