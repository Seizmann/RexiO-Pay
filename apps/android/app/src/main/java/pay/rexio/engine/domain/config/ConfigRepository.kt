package pay.rexio.engine.domain.config

import java.io.IOException
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.flow.first
import kotlinx.serialization.json.Json
import pay.rexio.engine.BuildConfig
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.remote.ApiException
import pay.rexio.engine.data.remote.ConfigResponseDto
import pay.rexio.engine.data.remote.DeviceApiProvider
import pay.rexio.engine.data.remote.apiCall

sealed interface ConfigSyncResult {
    data class Success(
        val config: ConfigResponseDto,
        val decision: UpdateDecision,
    ) : ConfigSyncResult

    data class Error(val exception: ApiException) : ConfigSyncResult
    data object NetworkError : ConfigSyncResult
}

/**
 * Fetches GET /v1/device/config: caches the sender-ID allowlist used by both
 * ingest paths and produces the update decision for the in-app update flow.
 */
@Singleton
class ConfigRepository @Inject constructor(
    private val apiProvider: DeviceApiProvider,
    private val prefsStore: PrefsStore,
    private val json: Json,
) {
    suspend fun sync(): ConfigSyncResult {
        val prefs = prefsStore.prefs.first()
        return try {
            val config = apiCall(json) { apiProvider.api(prefs.serverUrl).config() }
            prefsStore.updateSenderIds(config.senderIds)
            ConfigSyncResult.Success(
                config = config,
                decision = UpdatePolicy.evaluate(
                    currentVersion = BuildConfig.VERSION_NAME,
                    latestVersion = config.latestAppVersion,
                    mandatoryUpdate = config.mandatoryUpdate,
                    apkUrl = config.apkUrl,
                ),
            )
        } catch (e: ApiException) {
            ConfigSyncResult.Error(e)
        } catch (e: IOException) {
            ConfigSyncResult.NetworkError
        }
    }
}
