package pay.rexio.engine.core.sms

/**
 * Matches SMS senders against the sender-ID list downloaded from
 * GET /v1/device/config (sender_ids: provider → [display names]).
 * Unmatched senders are ignored by the ingest paths.
 */
object SenderMatcher {
    /** Fallback used until the first config fetch lands after pairing. */
    val DEFAULT_SENDER_IDS: Map<String, List<String>> = mapOf(
        "bkash" to listOf("bKash", "16247"),
        "nagad" to listOf("NAGAD", "16167"),
    )

    fun isConfigured(sender: String, senderIds: Map<String, List<String>>): Boolean =
        providerFor(sender, senderIds) != null

    /** Returns the provider key (e.g. "bkash") the sender belongs to, or null. */
    fun providerFor(sender: String, senderIds: Map<String, List<String>>): String? {
        val trimmed = sender.trim()
        if (trimmed.isEmpty()) return null
        return senderIds.entries.firstOrNull { (_, names) ->
            names.any { it.trim().equals(trimmed, ignoreCase = true) }
        }?.key
    }
}
