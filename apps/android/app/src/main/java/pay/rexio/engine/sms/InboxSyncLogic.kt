package pay.rexio.engine.sms

import pay.rexio.engine.core.sms.SenderMatcher

data class InboxRow(
    val sender: String,
    val body: String,
    val date: Long,
    val simSlot: Int?,
)

/** Pure decision logic for the inbox-sync fallback, kept free of Android APIs. */
object InboxSyncLogic {
    fun shouldIngest(row: InboxRow, senderIds: Map<String, List<String>>): Boolean =
        SenderMatcher.isConfigured(row.sender, senderIds)

    /**
     * The cursor always advances past every scanned row — matched or not —
     * otherwise unmatched SMS would be rescanned on every cycle.
     */
    fun nextCursor(rows: List<InboxRow>, currentCursor: Long): Long =
        rows.maxOfOrNull { it.date } ?: currentCursor
}
