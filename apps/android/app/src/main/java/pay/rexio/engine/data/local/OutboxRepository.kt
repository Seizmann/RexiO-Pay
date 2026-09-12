package pay.rexio.engine.data.local

import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class OutboxRepository @Inject constructor(
    private val dao: OutboxDao,
) {
    /**
     * Inserts a received SMS unless an identical one (same sender + body within
     * the dedupe window) is already queued — this also makes the dual ingest
     * paths (broadcast + inbox sync) safe against double-insertion.
     *
     * @return true when inserted, false when suppressed as a duplicate.
     */
    suspend fun enqueue(
        sender: String,
        body: String,
        simSlot: Int?,
        receivedAt: Long,
        provider: String? = null,
    ): Boolean {
        val trimmedSender = sender.trim()
        val trimmedBody = body.trim()
        if (trimmedBody.isEmpty()) return false
        val duplicates = dao.countDuplicates(
            sender = trimmedSender,
            body = trimmedBody,
            windowStart = receivedAt - DEDUPE_WINDOW_MS,
            windowEnd = receivedAt + DEDUPE_WINDOW_MS,
        )
        if (duplicates > 0) return false
        dao.insert(
            OutboxEntity(
                sender = trimmedSender,
                rawBody = trimmedBody,
                simSlot = simSlot,
                receivedAt = receivedAt,
                provider = provider,
            ),
        )
        return true
    }

    suspend fun pendingBatch(limit: Int = BATCH_LIMIT, now: Long = System.currentTimeMillis()) =
        dao.pendingOldestFirst(limit = limit, now = now)

    suspend fun markAttempted(ids: List<Long>) = dao.incrementAttempts(ids)

    suspend fun cooldown(ids: List<Long>, until: Long) = dao.setCooldown(ids, until)

    suspend fun deleteAcked(ids: List<Long>) = dao.deleteByIds(ids)

    fun pendingCount() = dao.pendingCount()

    suspend fun pendingCountOnce() = dao.pendingCountOnce()

    companion object {
        /** Same sender + body within this window is treated as one SMS. */
        const val DEDUPE_WINDOW_MS = 60_000L

        /** Max rows per POST /v1/device/sms request (server limit is 100). */
        const val BATCH_LIMIT = 100

        /** How long a row rests after exhausting retries before it is picked again. */
        const val COOLDOWN_MS = 6 * 60 * 60 * 1000L
    }
}
