package pay.rexio.engine.ui.settings

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.security.SecretStorage
import pay.rexio.engine.domain.config.ConfigRepository
import pay.rexio.engine.domain.config.ConfigSyncResult

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val prefsStore: PrefsStore,
    private val sessionStore: SessionStore,
    private val secretStorage: SecretStorage,
    private val configRepository: ConfigRepository,
) : ViewModel() {
    val serverUrl: StateFlow<String> = prefsStore.prefs
        .map { it.serverUrl }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), "")

    var updateResult by mutableStateOf<String?>(null)
        private set

    fun saveServerUrl(url: String, onSaved: () -> Unit) {
        val normalized = url.trim().trimEnd('/')
        if (!normalized.startsWith("https://") && !normalized.startsWith("http://")) {
            updateResult = "Enter a valid http(s) URL"
            return
        }
        viewModelScope.launch {
            prefsStore.setServerUrl(normalized)
            onSaved()
        }
    }

    fun checkForUpdates() {
        updateResult = "Checking…"
        viewModelScope.launch {
            updateResult = when (val result = configRepository.sync()) {
                is ConfigSyncResult.Success -> {
                    val decision = result.decision
                    when {
                        decision.updateAvailable -> "Update ${decision.latestVersion} available" +
                            (if (decision.mandatory) " (mandatory)" else "")
                        else -> "You are on the latest version"
                    }
                }
                is ConfigSyncResult.Error -> "Update check failed: ${result.exception.code}"
                ConfigSyncResult.NetworkError -> "Update check failed: no connection"
            }
        }
    }

    /**
     * Removes all pairing state (device id, secret, per-pairing prefs) — the
     * device must be re-paired with a fresh dashboard QR to work again.
     */
    fun unpair(onDone: () -> Unit) {
        viewModelScope.launch {
            sessionStore.clear()
            secretStorage.clear()
            prefsStore.resetState()
            onDone()
        }
    }
}
