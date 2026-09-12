package pay.rexio.engine.domain.sms

import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.flow.first
import pay.rexio.engine.core.sms.SenderMatcher
import pay.rexio.engine.data.local.OutboxRepository
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.work.UploadScheduler

/**
 * Single entry point for both ingest paths (broadcast receiver + inbox sync).
 * Applies the sender filter (unmatched senders are dropped), dedupes against
 * the outbox, and schedules the upload worker.
 */
@Singleton
class SmsIngestService @Inject constructor(
    private val outboxRepository: OutboxRepository,
    private val prefsStore: PrefsStore,
    private val sessionStore: SessionStore,
    private val uploadScheduler: UploadScheduler,
) {
    /** @return true when the SMS was accepted into the outbox. */
    suspend fun ingest(sender: String, body: String, simSlot: Int?, receivedAt: Long): Boolean {
        if (!sessionStore.current.isPaired) return false
        val senderIds = prefsStore.prefs.first().senderIds
            .ifEmpty { SenderMatcher.DEFAULT_SENDER_IDS }
        val provider = SenderMatcher.providerFor(sender, senderIds) ?: return false
        val inserted = outboxRepository.enqueue(sender, body, simSlot, receivedAt, provider)
        if (inserted) uploadScheduler.enqueueUpload()
        return inserted
    }
}
