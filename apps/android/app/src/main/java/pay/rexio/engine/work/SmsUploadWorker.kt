package pay.rexio.engine.work

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkerParameters
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.io.IOException
import java.time.Instant
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.flow.first
import kotlinx.serialization.json.Json
import pay.rexio.engine.data.local.OutboxRepository
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.remote.ApiException
import pay.rexio.engine.data.remote.DeviceApiProvider
import pay.rexio.engine.data.remote.SmsItemDto
import pay.rexio.engine.data.remote.SmsRequestDto
import pay.rexio.engine.data.remote.apiCall

/**
 * Uploads the oldest pending outbox rows in one batch (≤100, chronological) to
 * POST /v1/device/sms and deletes the rows echoed in the ack. Rows whose
 * client_id appears in the ack are deleted regardless of parsed/failed status —
 * the server has persisted the raw SMS either way. Known caveat: a retried
 * batch whose response was lost can be inserted twice server-side; the server
 * does not dedupe by client_id.
 */
@HiltWorker
class SmsUploadWorker @AssistedInject constructor(
    @Assisted context: Context,
    @Assisted params: WorkerParameters,
    private val outboxRepository: OutboxRepository,
    private val sessionStore: SessionStore,
    private val prefsStore: PrefsStore,
    private val apiProvider: DeviceApiProvider,
    private val json: Json,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        if (!sessionStore.current.isPaired) return Result.success()

        val now = System.currentTimeMillis()
        val prefs = prefsStore.prefs.first()

        if (prefs.authCooldownUntil > now) {
            // A 401 recently disabled signed traffic; wait for the cooldown to
            // expire rather than burning the server's 3-strike disable.
            return Result.failure()
        }

        val batch = outboxRepository.pendingBatch(now = now)
        if (batch.isEmpty()) return Result.success()

        if (UploadRetryPolicy.exceeded(runAttemptCount)) {
            outboxRepository.cooldown(batch.map { it.id }, now + OutboxRepository.COOLDOWN_MS)
            return Result.failure()
        }

        outboxRepository.markAttempted(batch.map { it.id })

        val request = SmsRequestDto(
            messages = batch.map { row ->
                SmsItemDto(
                    clientId = row.id.toString(),
                    sender = row.sender,
                    body = row.rawBody,
                    smsTime = Instant.ofEpochMilli(row.receivedAt).toString(),
                    simSlot = row.simSlot,
                    provider = row.provider,
                )
            },
        )

        return try {
            val response = apiCall(json) { apiProvider.api(prefs.serverUrl).sendSms(request) }
            val ackedIds = response.acks.mapNotNull { it.clientId.toLongOrNull() }
            outboxRepository.deleteAcked(ackedIds)
            prefsStore.setLastSyncAt(now)
            Result.success()
        } catch (e: ApiException) {
            when {
                e.httpStatus == 429 || e.httpStatus >= 500 -> Result.retry()
                e.httpStatus == 401 -> {
                    prefsStore.setAuthCooldownUntil(now + AUTH_COOLDOWN_MS)
                    Result.failure()
                }
                else -> Result.failure()
            }
        } catch (e: IOException) {
            Result.retry()
        }
    }

    companion object {
        const val NAME = "sms_upload"
        const val AUTH_COOLDOWN_MS = 10 * 60 * 1000L

        fun oneTimeRequest() = OneTimeWorkRequestBuilder<SmsUploadWorker>()
            .setConstraints(
                Constraints.Builder()
                    .setRequiredNetworkType(NetworkType.CONNECTED)
                    .build(),
            )
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 10, TimeUnit.SECONDS)
            .build()
    }
}
