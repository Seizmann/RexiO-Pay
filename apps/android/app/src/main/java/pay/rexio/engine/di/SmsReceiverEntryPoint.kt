package pay.rexio.engine.di

import dagger.hilt.EntryPoint
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import pay.rexio.engine.sms.SmsReceiver

/**
 * Members-injection entry point for the dynamically registered copy of
 * SmsReceiver. @AndroidEntryPoint receivers support members injection but are
 * not provider-bound, so the heartbeat service injects one via this.
 */
@EntryPoint
@InstallIn(SingletonComponent::class)
interface SmsReceiverEntryPoint {
    fun inject(receiver: SmsReceiver)
}
