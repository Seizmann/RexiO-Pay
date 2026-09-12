package pay.rexio.engine.sms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.provider.Telephony
import android.telephony.SmsMessage
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import pay.rexio.engine.core.sms.SimSlotExtractor
import pay.rexio.engine.core.sms.toValueMap
import pay.rexio.engine.domain.sms.SmsIngestService

data class ReceivedSms(
    val sender: String,
    val body: String,
    val simSlot: Int?,
    val receivedAt: Long,
)

/**
 * Manifest-registered receiver for SMS_RECEIVED. The heartbeat foreground
 * service additionally registers it dynamically — the outbox dedupe makes the
 * overlap safe.
 */
@AndroidEntryPoint
class SmsReceiver : BroadcastReceiver() {
    @Inject
    lateinit var ingestService: SmsIngestService

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) return
        val sms = parse(intent.extras) ?: return
        val pending = goAsync()
        CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
            try {
                ingestService.ingest(sms.sender, sms.body, sms.simSlot, sms.receivedAt)
            } finally {
                pending.finish()
            }
        }
    }

    internal fun parse(extras: Bundle?): ReceivedSms? {
        if (extras == null) return null
        val pdus = extras.get("pdus") as? Array<*> ?: return null
        val format = extras.getString("format")
        val messages = pdus.mapNotNull { pdu ->
            val bytes = pdu as? ByteArray ?: return@mapNotNull null
            runCatching { SmsMessage.createFromPdu(bytes, format) }
                .getOrElse { runCatching { SmsMessage.createFromPdu(bytes) }.getOrNull() }
        }
        if (messages.isEmpty()) return null
        val sender = messages.firstOrNull { it.displayOriginatingAddress != null }
            ?.displayOriginatingAddress ?: return null
        // Multipart: concatenate parts in array order (the reference app's
        // string-append was fine here; we keep it but via the PDU order).
        val body = messages.joinToString("") { it.displayMessageBody }
        return ReceivedSms(
            sender = sender,
            body = body,
            simSlot = SimSlotExtractor.extract(extras.toValueMap()),
            receivedAt = messages.first().timestampMillis,
        )
    }
}
