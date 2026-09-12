package pay.rexio.engine.core.crypto

import com.google.common.truth.Truth.assertThat
import org.junit.Test

class HmacSignerTest {
    // Same raw 32-byte secret, timestamp, and bodies used to generate the
    // expected signatures with the Go stdlib (backend's crypto stack).
    private val secret = "0123456789abcdef0123456789abcdef".toByteArray(Charsets.UTF_8)

    @Test
    fun `signs body vector identically to backend VerifySignature`() {
        val body = """{"messages":[]}"""
        assertThat(HmacSigner.sign(secret, 1_789_000_000L, body))
            .isEqualTo("5f96f3cad846a34692387896fed8fb16fd78b6b17a2d2f56afe579f35302af17")
    }

    @Test
    fun `signs empty body as timestamp plus trailing dot`() {
        assertThat(HmacSigner.sign(secret, 1_789_000_000L, ""))
            .isEqualTo("aaffcaed1b5199d2ac34b344252a8c3479c2da2ba60a691e565f0ab8a0613ea3")
    }

    @Test
    fun `canonical string is timestamp dot body`() {
        assertThat(HmacSigner.canonicalString(123, "abc")).isEqualTo("123.abc")
    }

    @Test
    fun `decodes unpadded base64url secret back to the original 32 bytes`() {
        val decoded = HmacSigner.decodeSecret("AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8")
        assertThat(decoded).hasLength(32)
        assertThat(decoded[0].toInt()).isEqualTo(0)
        assertThat(decoded[31].toInt()).isEqualTo(31)
    }
}
