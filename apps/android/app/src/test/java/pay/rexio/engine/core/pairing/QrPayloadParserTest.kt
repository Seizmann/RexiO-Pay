package pay.rexio.engine.core.pairing

import com.google.common.truth.Truth.assertThat
import org.junit.Test

class QrPayloadParserTest {
    @Test
    fun `parses canonical json payload`() {
        val result = QrPayloadParser.parse(
            """{"server_url":"https://api.pay.rexio.pro","merchant_id":"mer_abc","pairing_token":"tok123"}""",
        )
        val payload = (result as QrParseResult.Success).payload
        assertThat(payload.serverUrl).isEqualTo("https://api.pay.rexio.pro")
        assertThat(payload.merchantId).isEqualTo("mer_abc")
        assertThat(payload.pairingToken).isEqualTo("tok123")
    }

    @Test
    fun `ignores unknown json fields`() {
        val result = QrPayloadParser.parse(
            """{"server_url":"https://x.example","merchant_id":"m","pairing_token":"t","v":2}""",
        )
        assertThat(result).isInstanceOf(QrParseResult.Success::class.java)
    }

    @Test
    fun `parses uri form with url encoded token`() {
        val result = QrPayloadParser.parse(
            "rexio-pay://pair?server_url=https%3A%2F%2Fapi.pay.rexio.pro&merchant_id=mer_1&pairing_token=a%2Bb%3Dc",
        )
        val payload = (result as QrParseResult.Success).payload
        assertThat(payload.serverUrl).isEqualTo("https://api.pay.rexio.pro")
        assertThat(payload.pairingToken).isEqualTo("a+b=c")
    }

    @Test
    fun `rejects non https server urls`() {
        val result = QrPayloadParser.parse(
            """{"server_url":"http://api.pay.rexio.pro","merchant_id":"m","pairing_token":"t"}""",
        )
        assertThat((result as QrParseResult.Failure).reason).contains("https")
    }

    @Test
    fun `rejects payloads with missing fields`() {
        val missingMerchant = QrPayloadParser.parse(
            """{"server_url":"https://x.example","pairing_token":"t"}""",
        )
        assertThat(missingMerchant).isInstanceOf(QrParseResult.Failure::class.java)

        val missingToken = QrPayloadParser.parse(
            "rexio-pay://pair?server_url=https://x.example&merchant_id=m",
        )
        assertThat(missingToken).isInstanceOf(QrParseResult.Failure::class.java)
    }

    @Test
    fun `rejects non payload content`() {
        assertThat(QrPayloadParser.parse("")).isInstanceOf(QrParseResult.Failure::class.java)
        assertThat(QrPayloadParser.parse("hello world")).isInstanceOf(QrParseResult.Failure::class.java)
        assertThat(QrPayloadParser.parse("https://example.com/not-a-pairing")).isInstanceOf(QrParseResult.Failure::class.java)
    }
}
