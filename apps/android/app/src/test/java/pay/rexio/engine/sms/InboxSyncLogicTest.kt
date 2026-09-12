package pay.rexio.engine.sms

import com.google.common.truth.Truth.assertThat
import org.junit.Test

class InboxSyncLogicTest {
    private val senderIds = mapOf(
        "bkash" to listOf("bKash", "16247"),
        "nagad" to listOf("NAGAD", "16167"),
    )

    @Test
    fun `ingests only configured senders`() {
        assertThat(InboxSyncLogic.shouldIngest(row("bKash"), senderIds)).isTrue()
        assertThat(InboxSyncLogic.shouldIngest(row("16247"), senderIds)).isTrue()
        assertThat(InboxSyncLogic.shouldIngest(row("Robi"), senderIds)).isFalse()
    }

    @Test
    fun `cursor advances past every scanned row including unmatched`() {
        val rows = listOf(
            row("Robi", date = 1_000L),
            row("bKash", date = 2_000L),
            row("Grameenphone", date = 3_000L),
        )
        assertThat(InboxSyncLogic.nextCursor(rows, 500L)).isEqualTo(3_000L)
    }

    @Test
    fun `cursor stays put when nothing new was scanned`() {
        assertThat(InboxSyncLogic.nextCursor(emptyList(), 5_000L)).isEqualTo(5_000L)
    }

    private fun row(sender: String, date: Long = 0L) =
        InboxRow(sender = sender, body = "body", date = date, simSlot = null)
}
