package pay.rexio.engine.data.remote

import com.google.common.truth.Truth.assertThat
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.junit.After
import org.junit.Before
import org.junit.Test
import pay.rexio.engine.core.crypto.HmacSigner
import pay.rexio.engine.core.time.Clock

class HmacAuthInterceptorTest {
    private val secret = "0123456789abcdef0123456789abcdef".toByteArray(Charsets.UTF_8)
    private val fixedMillis = 1_789_000_000_000L // → 1789000000 seconds
    private lateinit var server: MockWebServer
    private lateinit var client: OkHttpClient

    private val clock = object : Clock {
        override fun nowMillis() = fixedMillis
    }

    private fun clientWith(auth: AuthSource): OkHttpClient =
        OkHttpClient.Builder().addInterceptor(HmacAuthInterceptor(auth, clock)).build()

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        client = clientWith(
            object : AuthSource {
                override fun deviceId() = "dev_test1234567890123456789"
                override fun deviceSecret() = secret
            },
        )
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `signs post requests over the exact raw body`() {
        val body = """{"messages":[{"client_id":"7","sender":"bKash","body":"Tk 100"}]}"""
        server.enqueue(MockResponse().setResponseCode(200).setBody("{}"))
        client.newCall(
            Request.Builder()
                .url(server.url("/v1/device/sms"))
                .post(body.toRequestBody("application/json".toMediaType()))
                .build(),
        ).execute().close()

        val recorded = server.takeRequest()
        assertThat(recorded.getHeader("X-Rexio-Device-Id")).isEqualTo("dev_test1234567890123456789")
        assertThat(recorded.getHeader("X-Rexio-Timestamp")).isEqualTo("1789000000")
        assertThat(recorded.getHeader("X-Rexio-Signature"))
            .isEqualTo(HmacSigner.sign(secret, 1_789_000_000L, body))
        assertThat(recorded.body.readUtf8()).isEqualTo(body)
    }

    @Test
    fun `signs get requests over the empty body`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("{}"))
        client.newCall(Request.Builder().url(server.url("/v1/device/config")).get().build())
            .execute().close()

        val recorded = server.takeRequest()
        // Cross-checked against the Go stdlib: HMAC-SHA256(secret, "1789000000.")
        assertThat(recorded.getHeader("X-Rexio-Signature"))
            .isEqualTo("aaffcaed1b5199d2ac34b344252a8c3479c2da2ba60a691e565f0ab8a0613ea3")
    }

    @Test
    fun `passes through unsigned when device id is missing`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("{}"))
        val unsigned = clientWith(
            object : AuthSource {
                override fun deviceId() = null
                override fun deviceSecret() = secret
            },
        )
        unsigned.newCall(
            Request.Builder()
                .url(server.url("/v1/device/pair"))
                .post("{}".toRequestBody("application/json".toMediaType()))
                .build(),
        ).execute().close()

        val recorded = server.takeRequest()
        assertThat(recorded.getHeader("X-Rexio-Device-Id")).isNull()
        assertThat(recorded.getHeader("X-Rexio-Signature")).isNull()
    }

    @Test
    fun `passes through unsigned when secret is missing`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("{}"))
        val unsigned = clientWith(
            object : AuthSource {
                override fun deviceId() = "dev_test1234567890123456789"
                override fun deviceSecret() = null
            },
        )
        unsigned.newCall(
            Request.Builder()
                .url(server.url("/v1/device/heartbeat"))
                .post("{}".toRequestBody("application/json".toMediaType()))
                .build(),
        ).execute().close()

        val recorded = server.takeRequest()
        assertThat(recorded.getHeader("X-Rexio-Signature")).isNull()
    }
}
