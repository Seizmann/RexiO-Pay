package pay.rexio.engine.sms

import android.content.ContentResolver
import android.content.Context
import android.net.Uri
import android.provider.Telephony
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import androidx.work.Constraints
import androidx.work.NetworkType
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.flow.first
import pay.rexio.engine.core.sms.SenderMatcher
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.domain.sms.SmsIngestService

/**
 * Fallback ingest path: queries content://sms/inbox for SMS newer than the
 * stored cursor and queues the matching ones. Runs periodically (15 min) and
 * as a one-shot on network reconnect; the broadcast path remains the primary
 * one.
 */
@HiltWorker
class InboxSyncWorker @AssistedInject constructor(
    @Assisted context: Context,
    @Assisted params: WorkerParameters,
    private val ingestService: SmsIngestService,
    private val prefsStore: PrefsStore,
    private val sessionStore: SessionStore,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        if (!sessionStore.current.isPaired) return Result.success()
        val prefs = prefsStore.prefs.first()
        val senderIds = prefs.senderIds.ifEmpty { SenderMatcher.DEFAULT_SENDER_IDS }
        val rows = queryInbox(prefs.inboxScanCursor).sortedBy { it.date }
        for (row in rows) {
            if (InboxSyncLogic.shouldIngest(row, senderIds)) {
                ingestService.ingest(row.sender, row.body, row.simSlot, row.date)
            }
        }
        prefsStore.setInboxScanCursor(InboxSyncLogic.nextCursor(rows, prefs.inboxScanCursor))
        return Result.success()
    }

    private fun queryInbox(cursorMillis: Long): List<InboxRow> {
        val uri = Telephony.Sms.Inbox.CONTENT_URI
        val where = "${Telephony.Sms.Inbox.DATE} > ?"
        val args = arrayOf(cursorMillis.toString())
        val standard = arrayOf(
            Telephony.Sms.Inbox.ADDRESS,
            Telephony.Sms.Inbox.BODY,
            Telephony.Sms.Inbox.DATE,
        )
        // Some ROMs expose the receiving slot in a non-standard sim_id column;
        // include it when present and fall back to the standard projection.
        val withSim = standard + "sim_id"
        return runCatching { query(uri, withSim, where, args, hasSimId = true) }
            .getOrElse { query(uri, standard, where, args, hasSimId = false) }
    }

    private fun query(
        uri: Uri,
        projection: Array<String>,
        where: String,
        args: Array<String>,
        hasSimId: Boolean,
    ): List<InboxRow> {
        val resolver: ContentResolver = applicationContext.contentResolver
        val rows = mutableListOf<InboxRow>()
        resolver.query(uri, projection, where, args, null)?.use { cursor ->
            val addressIdx = cursor.getColumnIndexOrThrow(Telephony.Sms.Inbox.ADDRESS)
            val bodyIdx = cursor.getColumnIndexOrThrow(Telephony.Sms.Inbox.BODY)
            val dateIdx = cursor.getColumnIndexOrThrow(Telephony.Sms.Inbox.DATE)
            val simIdx = if (hasSimId) cursor.getColumnIndex("sim_id") else -1
            while (cursor.moveToNext()) {
                rows += InboxRow(
                    sender = cursor.getString(addressIdx) ?: "",
                    body = cursor.getString(bodyIdx) ?: "",
                    date = cursor.getLong(dateIdx),
                    simSlot = if (simIdx >= 0) {
                        cursor.getLong(simIdx).toInt().takeIf { it >= 0 }
                    } else {
                        null
                    },
                )
            }
        }
        return rows
    }

    companion object {
        const val PERIODIC_NAME = "inbox_sync_periodic"
        const val ONE_TIME_NAME = "inbox_sync_once"
        const val INTERVAL_MINUTES = 15L

        fun schedulePeriodic(context: Context) {
            WorkManager.getInstance(context).enqueueUniquePeriodicWork(
                PERIODIC_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                PeriodicWorkRequestBuilder<InboxSyncWorker>(INTERVAL_MINUTES, TimeUnit.MINUTES)
                    .setConstraints(networkConstraint())
                    .build(),
            )
        }

        fun scheduleOnce(context: Context) {
            WorkManager.getInstance(context).enqueueUniqueWork(
                ONE_TIME_NAME,
                ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<InboxSyncWorker>()
                    .setConstraints(networkConstraint())
                    .build(),
            )
        }

        private fun networkConstraint() = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()
    }
}
