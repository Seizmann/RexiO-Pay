package pay.rexio.engine.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.ServiceInfo
import android.net.ConnectivityManager
import android.net.Network
import android.os.Build
import android.os.IBinder
import android.provider.Telephony
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import dagger.hilt.android.AndroidEntryPoint
import dagger.hilt.android.EntryPointAccessors
import javax.inject.Inject
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import pay.rexio.engine.R
import pay.rexio.engine.data.local.OutboxRepository
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.di.SmsReceiverEntryPoint
import pay.rexio.engine.domain.config.ConfigRepository
import pay.rexio.engine.domain.heartbeat.HeartbeatEngine
import pay.rexio.engine.domain.heartbeat.HeartbeatStatus
import pay.rexio.engine.sms.InboxSyncWorker
import pay.rexio.engine.sms.SmsReceiver
import pay.rexio.engine.ui.MainActivity
import pay.rexio.engine.work.UploadScheduler

/**
 * Persistent foreground service: heartbeat every 60 s, upload scheduling when
 * the queue is non-empty, periodic config refresh, dynamic SMS receiver
 * (belt to the manifest receiver's braces), and a network callback that kicks
 * the inbox-sync fallback on reconnect. The persistent notification doubles as
 * the anti-kill story on aggressive OEMs.
 */
@AndroidEntryPoint
class HeartbeatService : Service() {
    @Inject lateinit var heartbeatEngine: HeartbeatEngine
    @Inject lateinit var heartbeatStatus: HeartbeatStatus
    @Inject lateinit var sessionStore: SessionStore
    @Inject lateinit var configRepository: ConfigRepository
    @Inject lateinit var outboxRepository: OutboxRepository
    @Inject lateinit var uploadScheduler: UploadScheduler

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var networkCallback: ConnectivityManager.NetworkCallback? = null
    private var lastConfigSyncAt = 0L

    /** Second receiver instance (belt to the manifest one), Hilt-injected lazily. */
    private val dynamicSmsReceiver: SmsReceiver by lazy {
        SmsReceiver().also { receiver ->
            EntryPointAccessors.fromApplication(
                applicationContext,
                SmsReceiverEntryPoint::class.java,
            ).inject(receiver)
        }
    }

    override fun onCreate() {
        super.onCreate()
        startAsForeground()
        registerDynamicSmsReceiver()
        registerNetworkCallback()
        InboxSyncWorker.schedulePeriodic(this)
        scope.launch { loop() }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int = START_STICKY

    override fun onDestroy() {
        networkCallback?.let {
            runCatching {
                getSystemService(ConnectivityManager::class.java).unregisterNetworkCallback(it)
            }
        }
        runCatching { unregisterReceiver(dynamicSmsReceiver) }
        scope.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun startAsForeground() {
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(
            NotificationChannel(
                CHANNEL_ID,
                "RexiO Pay heartbeat",
                NotificationManager.IMPORTANCE_LOW,
            ).apply {
                description = "Keeps this device online for payment SMS forwarding"
            },
        )
        val launchIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.app_name))
            .setContentText("Forwarding payment SMS")
            .setSmallIcon(R.drawable.ic_stat_forward)
            .setOngoing(true)
            .setContentIntent(launchIntent)
            .build()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(
                NOTIFICATION_ID,
                notification,
                ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC,
            )
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun registerDynamicSmsReceiver() {
        val filter = IntentFilter(Telephony.Sms.Intents.SMS_RECEIVED_ACTION)
        ContextCompat.registerReceiver(
            this,
            dynamicSmsReceiver,
            filter,
            ContextCompat.RECEIVER_EXPORTED,
        )
    }

    private fun registerNetworkCallback() {
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                InboxSyncWorker.scheduleOnce(applicationContext)
            }
        }
        networkCallback = callback
        runCatching {
            getSystemService(ConnectivityManager::class.java)
                .registerDefaultNetworkCallback(callback)
        }
    }

    private suspend fun loop() {
        while (scope.isActive) {
            delay(HEARTBEAT_INTERVAL_MS)
            if (!sessionStore.current.isPaired) continue

            heartbeatStatus.update(heartbeatEngine.heartbeatOnce())

            if (outboxRepository.pendingCountOnce() > 0) {
                uploadScheduler.enqueueUpload()
            }

            val now = System.currentTimeMillis()
            if (now - lastConfigSyncAt >= CONFIG_INTERVAL_MS) {
                lastConfigSyncAt = now
                configRepository.sync()
            }
        }
    }

    companion object {
        const val CHANNEL_ID = "rexio_heartbeat"
        const val NOTIFICATION_ID = 1
        private const val HEARTBEAT_INTERVAL_MS = 60_000L
        private const val CONFIG_INTERVAL_MS = 30 * 60_000L

        fun start(context: Context) {
            ContextCompat.startForegroundService(
                context,
                Intent(context, HeartbeatService::class.java),
            )
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, HeartbeatService::class.java))
        }
    }
}
