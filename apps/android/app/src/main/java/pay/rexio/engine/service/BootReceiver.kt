package pay.rexio.engine.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject
import pay.rexio.engine.data.prefs.SessionStore

/** Restarts the heartbeat service after reboot or an app update, when paired. */
@AndroidEntryPoint
class BootReceiver : BroadcastReceiver() {
    @Inject
    lateinit var sessionStore: SessionStore

    override fun onReceive(context: Context, intent: Intent) {
        val action = intent.action
        if (action != Intent.ACTION_BOOT_COMPLETED &&
            action != Intent.ACTION_MY_PACKAGE_REPLACED
        ) {
            return
        }
        if (sessionStore.current.isPaired) {
            HeartbeatService.start(context)
        }
    }
}
