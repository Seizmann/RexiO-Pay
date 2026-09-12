package pay.rexio.engine.core.phone

import com.google.common.truth.Truth.assertThat
import com.google.common.truth.Truth.assertWithMessage
import org.junit.Test

class PhoneCanonicalizerTest {
    @Test
    fun `matches the backend canonicalization table exactly`() {
        val cases = mapOf(
            "01712345678" to "01712345678",
            "+8801712345678" to "01712345678",
            "8801712345678" to "01712345678",
            "88 01712345678" to "01712345678",
            "017-1234-5678" to "01712345678",
            "+880-017-1234-5678" to "01712345678",
            "880017-1234-5678" to "01712345678",
            "" to "",
            "1234567890" to "",
            "0171234567" to "", // 10 digits — too short
            "017123456789" to "", // 12 digits — too long
            "02712345678" to "", // does not start with 01
        )
        cases.forEach { (input, want) ->
            assertWithMessage("Canonicalize(%s)".format(input))
                .that(PhoneCanonicalizer.canonicalize(input))
                .isEqualTo(want)
        }
    }

    @Test
    fun `accepts only valid canonical numbers`() {
        assertThat(PhoneCanonicalizer.isValid("01712345678")).isTrue()
        assertThat(PhoneCanonicalizer.isValid("123")).isFalse()
    }
}
