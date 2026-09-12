package pay.rexio.engine.core.sms

import com.google.common.truth.Truth.assertThat
import org.junit.Test

class SenderMatcherTest {
    private val senderIds = mapOf(
        "bkash" to listOf("bKash", "16247"),
        "nagad" to listOf("NAGAD", "16167"),
    )

    @Test
    fun `matches display names case insensitively`() {
        assertThat(SenderMatcher.isConfigured("bkash", senderIds)).isTrue()
        assertThat(SenderMatcher.isConfigured("bKASH", senderIds)).isTrue()
        assertThat(SenderMatcher.isConfigured("  NAGAD ", senderIds)).isTrue()
    }

    @Test
    fun `matches numeric shortcodes`() {
        assertThat(SenderMatcher.isConfigured("16247", senderIds)).isTrue()
        assertThat(SenderMatcher.isConfigured("16167", senderIds)).isTrue()
    }

    @Test
    fun `derives provider from the matched sender`() {
        assertThat(SenderMatcher.providerFor("bKash", senderIds)).isEqualTo("bkash")
        assertThat(SenderMatcher.providerFor("16167", senderIds)).isEqualTo("nagad")
    }

    @Test
    fun `rejects unknown senders and blank input`() {
        assertThat(SenderMatcher.isConfigured("Robi", senderIds)).isFalse()
        assertThat(SenderMatcher.providerFor("Robi", senderIds)).isNull()
        assertThat(SenderMatcher.isConfigured("", senderIds)).isFalse()
        assertThat(SenderMatcher.isConfigured("  ", senderIds)).isFalse()
    }

    @Test
    fun `default sender ids cover bkash and nagad`() {
        assertThat(SenderMatcher.isConfigured("bKash", SenderMatcher.DEFAULT_SENDER_IDS)).isTrue()
        assertThat(SenderMatcher.isConfigured("NAGAD", SenderMatcher.DEFAULT_SENDER_IDS)).isTrue()
    }
}
