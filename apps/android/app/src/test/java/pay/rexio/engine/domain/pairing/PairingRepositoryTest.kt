package pay.rexio.engine.domain.pairing

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.google.common.truth.Truth.assertThat
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
import pay.rexio.engine.core.pairing.QrPayload
import pay.rexio.engine.core.sim.DeviceInfo
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.remote.DeviceApiProvider
import pay.rexio.engine.data.security.SecretStorage

/** In-memory stand-in: the real Android Keystore does not exist on the JVM. */
private class FakeSecretStorage : SecretStorage {
    var stored: String? = null
    override fun deviceSecret(): ByteArray? = stored?.let {
        java.util.Base64.getUrlDecoder().decode(it)
    }
    override fun saveDeviceSecret(base64UrlSecret: String) { stored = base64UrlSecret }
    override fun clear() { stored = null }
}

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PairingRepositoryTest {
    private lateinit var server: MockWebServer
    private lateinit var session: SessionStore
    private lateinit var secret: FakeSecretStorage
    private lateinit var prefs: PrefsStore
    private lateinit var repository: PairingRepository

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
        session = SessionStore(context)
        secret = FakeSecretStorage()
        prefs = PrefsStore(context, "pair_test_${System.nanoTime()}")
        repository = PairingRepository(
            apiProvider = DeviceApiProvider(OkHttpClient.Builder().build(), json),
            sessionStore = session,
            secretStorage = secret,
            prefsStore = prefs,
            json = json,
            deviceInfo = object : DeviceInfo {
                override val model = "Test Phone"
                override val androidVersion = "14"
                override val appVersion = "1.0.0"
            },
        )
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `pairing persists device id and keystore-backed secret`() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(200).setBody(
                """{"device_id":"dev_abc1234567890123456789","device_secret":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8","status":"active"}""",
            ),
        )
        val result = repository.pair(
            QrPayload(server.url("/").toString().removeSuffix("/"), "mer_1", "tok123"),
        )
        assertThat(result).isInstanceOf(PairResult.Success::class.java)
        assertThat(session.current.deviceId).isEqualTo("dev_abc1234567890123456789")
        assertThat(session.current.merchantId).isEqualTo("mer_1")
        assertThat(secret.deviceSecret()).hasLength(32)

        val recorded = server.takeRequest()
        assertThat(recorded.path).isEqualTo("/v1/device/pair")
        val body = recorded.body.readUtf8()
        assertThat(body).contains("\"pairing_token\":\"tok123\"")
        assertThat(body).contains("\"app_version\":\"1.0.0\"")
        assertThat(body).contains("\"model\":\"Test Phone\"")
    }

    @Test
    fun `rejected pairing surfaces the server error code without persisting`() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(401).setBody(
                """{"error":{"code":"unauthorized","message":"pairing token is invalid or expired"}}""",
            ),
        )
        val result = repository.pair(
            QrPayload(server.url("/").toString().removeSuffix("/"), "mer_1", "bad"),
        )
        assertThat(result).isInstanceOf(PairResult.Error::class.java)
        result as PairResult.Error
        assertThat(result.code).isEqualTo("unauthorized")
        assertThat(result.message).isEqualTo("pairing token is invalid or expired")
        assertThat(session.current.isPaired).isFalse()
        assertThat(secret.deviceSecret()).isNull()
    }

    @Test
    fun `unreachable server maps to a network error`() = runTest {
        val result = repository.pair(QrPayload("http://127.0.0.1:1", "mer_1", "tok"))
        assertThat(result).isInstanceOf(PairResult.Error::class.java)
        result as PairResult.Error
        assertThat(result.code).isEqualTo("network")
        assertThat(session.current.isPaired).isFalse()
    }
}
