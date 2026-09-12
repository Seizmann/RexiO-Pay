package pay.rexio.engine.domain.heartbeat

import java.io.IOException
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.flow.first
import kotlinx.serialization.json.Json
import pay.rexio.engine.BuildConfig
import pay.rexio.engine.data.local.OutboxRepository
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.remote.ApiException
import pay.rexio.engine.data.remote.DeviceApiProvider
import pay.rexio.engine.data.remote.HeartbeatRequestDto
import pay.rexio.engine.data.remote.apiCall
import pay.rexio.engine.work.SmsUploadWorker

sealed interface HeartbeatOutcome {
    data object Unpaired : HeartbeatOutcome
    data object AuthPaused : HeartbeatOutcome
    data object Ok : HeartbeatOutcome
    data class Failed(val cause: Exception) : HeartbeatOutcome
}

@Singleton
class HeartbeatEngine @Inject constructor(
    private val apiProvider: DeviceApiProvider,
    private val prefsStore: PrefsStore,
    private val sessionStore: SessionStore,
    private val outboxRepository: OutboxRepository,
    private val battery: BatteryLevelProvider,
    private val json: Json,
) {
    /**
     * One heartbeat tick. On a 401 the signing cooldown is engaged to protect
     * the server's 3-strike auto-disable from a bad device clock.
     */
    suspend fun heartbeatOnce(): HeartbeatOutcome {
        if (!sessionStore.current.isPaired) return HeartbeatOutcome.Unpaired
        val prefs = prefsStore.prefs.first()
        if (prefs.authCooldownUntil > System.currentTimeMillis()) return HeartbeatOutcome.AuthPaused
        return try {
            apiCall(json) {
                apiProvider.api(prefs.serverUrl).heartbeat(
                    HeartbeatRequestDto(
                        batteryLevel = battery.current(),
                        appVersion = BuildConfig.VERSION_NAME,
                        queueDepth = outboxRepository.pendingCountOnce(),
                    ),
                )
            }
            HeartbeatOutcome.Ok
        } catch (e: ApiException) {
            if (e.httpStatus == 401) {
                prefsStore.setAuthCooldownUntil(
                    System.currentTimeMillis() + SmsUploadWorker.AUTH_COOLDOWN_MS,
                )
            }
            HeartbeatOutcome.Failed(e)
        } catch (e: IOException) {
            HeartbeatOutcome.Failed(e)
        }
    }
}
