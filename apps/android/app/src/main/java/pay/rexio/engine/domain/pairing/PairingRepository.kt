package pay.rexio.engine.domain.pairing

import java.io.IOException
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.serialization.json.Json
import pay.rexio.engine.core.pairing.QrPayload
import pay.rexio.engine.core.sim.DeviceInfo
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.remote.ApiException
import pay.rexio.engine.data.remote.DeviceApiProvider
import pay.rexio.engine.data.remote.PairRequestDto
import pay.rexio.engine.data.remote.apiCall
import pay.rexio.engine.data.security.SecretStorage

sealed interface PairResult {
    data class Success(val deviceId: String) : PairResult
    data class Error(val code: String, val message: String) : PairResult
}

@Singleton
class PairingRepository @Inject constructor(
    private val apiProvider: DeviceApiProvider,
    private val sessionStore: SessionStore,
    private val secretStorage: SecretStorage,
    private val prefsStore: PrefsStore,
    private val json: Json,
    private val deviceInfo: DeviceInfo,
) {
    /**
     * Pairs the device: sends the QR's one-time token, then persists the
     * returned device_id and device_secret. The secret goes to Keystore-backed
     * storage only — it is never logged or held anywhere else.
     */
    suspend fun pair(payload: QrPayload): PairResult {
        return try {
            val response = apiCall(json) {
                apiProvider.api(payload.serverUrl).pair(
                    PairRequestDto(
                        pairingToken = payload.pairingToken,
                        deviceName = deviceInfo.model,
                        model = deviceInfo.model,
                        androidVersion = deviceInfo.androidVersion,
                        appVersion = deviceInfo.appVersion,
                    ),
                )
            }
            secretStorage.saveDeviceSecret(response.deviceSecret)
            sessionStore.setPaired(response.deviceId, payload.merchantId)
            prefsStore.setServerUrl(payload.serverUrl)
            prefsStore.resetState()
            PairResult.Success(response.deviceId)
        } catch (e: ApiException) {
            PairResult.Error(e.code, e.message ?: "Pairing failed")
        } catch (e: IOException) {
            PairResult.Error("network", "Could not reach ${payload.serverUrl}")
        }
    }
}
