package pay.rexio.engine.data.remote

import com.google.common.truth.Truth.assertThat
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import org.junit.After
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory

class DeviceApiTest {
    private lateinit var server: MockWebServer
    private lateinit var api: DeviceApi

    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        encodeDefaults = false
    }

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        api = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient.Builder().build())
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(DeviceApi::class.java)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `pair request and response round trip with exact field names`() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(200).setBody(
                """{"device_id":"dev_abc123","device_secret":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8","status":"active"}""",
            ),
        )
        val response = api.pair(
            PairRequestDto(
                pairingToken = "tok",
                deviceName = "Shop Phone",
                model = "Samsung A54",
                androidVersion = "14",
                appVersion = "1.0.0",
            ),
        )
        val recorded = server.takeRequest()
        assertThat(recorded.path).isEqualTo("/v1/device/pair")
        assertThat(recorded.body.readUtf8()).isEqualTo(
            """{"pairing_token":"tok","device_name":"Shop Phone","model":"Samsung A54","android_version":"14","app_version":"1.0.0"}""",
        )
        assertThat(response.deviceId).isEqualTo("dev_abc123")
        assertThat(response.deviceSecret).isEqualTo("AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8")
        assertThat(response.status).isEqualTo("active")
    }

    @Test
    fun `sms request omits null sim slot and provider`() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"acks":[]}"""))
        api.sendSms(
            SmsRequestDto(
                messages = listOf(
                    SmsItemDto(
                        clientId = "42",
                        sender = "bKash",
                        body = "You have received Tk 100",
                        smsTime = "2026-09-12T14:30:00Z",
                    ),
                ),
            ),
        )
        val recorded = server.takeRequest()
        assertThat(recorded.body.readUtf8()).isEqualTo(
            """{"messages":[{"client_id":"42","sender":"bKash","body":"You have received Tk 100","sms_time":"2026-09-12T14:30:00Z"}]}""",
        )
    }

    @Test
    fun `sms request includes sim slot and provider when present`() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"acks":[]}"""))
        api.sendSms(
            SmsRequestDto(
                messages = listOf(
                    SmsItemDto(
                        clientId = "7",
                        sender = "NAGAD",
                        body = "Money Received.",
                        simSlot = 1,
                        provider = "nagad",
                    ),
                ),
            ),
        )
        val recorded = server.takeRequest()
        assertThat(recorded.body.readUtf8()).isEqualTo(
            """{"messages":[{"client_id":"7","sender":"NAGAD","body":"Money Received.","sim_slot":1,"provider":"nagad"}]}""",
        )
    }

    @Test
    fun `sms acks parse and echo client ids`() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(200).setBody(
                """{"acks":[{"client_id":"1","sms_id":"sms_abc","status":"parsed"},{"client_id":"2","status":"failed"}]}""",
            ),
        )
        val response = api.sendSms(SmsRequestDto(messages = emptyList()))
        assertThat(response.acks).hasSize(2)
        assertThat(response.acks[0].clientId).isEqualTo("1")
        assertThat(response.acks[0].status).isEqualTo("parsed")
        assertThat(response.acks[1].smsId).isNull()
        assertThat(response.acks[1].status).isEqualTo("failed")
    }

    @Test
    fun `heartbeat round trip`() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"status":"ok"}"""))
        val response = api.heartbeat(
            HeartbeatRequestDto(batteryLevel = 87, appVersion = "1.0.0", queueDepth = 3),
        )
        val recorded = server.takeRequest()
        assertThat(recorded.path).isEqualTo("/v1/device/heartbeat")
        assertThat(recorded.body.readUtf8())
            .isEqualTo("""{"battery_level":87,"app_version":"1.0.0","queue_depth":3}""")
        assertThat(response.status).isEqualTo("ok")
    }

    @Test
    fun `config response parses sender ids and optional fields`() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(200).setBody(
                """{"latest_app_version":"1.3.0","apk_url":"https://example.com/a.apk","mandatory_update":false,"parser_version":"sms-v1","sender_ids":{"bkash":["bKash","16247"],"nagad":["NAGAD","16167"]}}""",
            ),
        )
        val config = api.config()
        assertThat(config.latestAppVersion).isEqualTo("1.3.0")
        assertThat(config.apkUrl).isEqualTo("https://example.com/a.apk")
        assertThat(config.mandatoryUpdate).isFalse()
        assertThat(config.senderIds["bkash"]).containsExactly("bKash", "16247")
    }

    @Test
    fun `config tolerates absent optional fields`() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"mandatory_update":true,"sender_ids":{}}"""))
        val config = api.config()
        assertThat(config.latestAppVersion).isNull()
        assertThat(config.mandatoryUpdate).isTrue()
        assertThat(config.senderIds).isEmpty()
    }

    @Test
    fun `error envelope maps to ApiException with code and message`() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(401).setBody(
                """{"error":{"code":"device_disabled","message":"device is disabled"}}""",
            ),
        )
        val exception = runCatching { apiCall(json) { api.config() } }.exceptionOrNull()
        assertThat(exception).isInstanceOf(ApiException::class.java)
        exception as ApiException
        assertThat(exception.code).isEqualTo("device_disabled")
        assertThat(exception.message).isEqualTo("device is disabled")
        assertThat(exception.httpStatus).isEqualTo(401)
    }

    @Test
    fun `non envelope error falls back to http status code`() = runTest {
        server.enqueue(MockResponse().setResponseCode(500).setBody("boom"))
        val exception = runCatching { apiCall(json) { api.config() } }.exceptionOrNull()
        assertThat(exception).isInstanceOf(ApiException::class.java)
        exception as ApiException
        assertThat(exception.code).isEqualTo("http_500")
        assertThat(exception.httpStatus).isEqualTo(500)
    }
}
