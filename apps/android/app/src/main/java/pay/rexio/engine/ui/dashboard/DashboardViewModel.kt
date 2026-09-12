package pay.rexio.engine.ui.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import pay.rexio.engine.data.local.OutboxRepository
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.prefs.SimProfile
import pay.rexio.engine.domain.config.ConfigRepository
import pay.rexio.engine.domain.config.ConfigSyncResult
import pay.rexio.engine.domain.config.UpdateDecision
import pay.rexio.engine.domain.heartbeat.HeartbeatOutcome
import pay.rexio.engine.domain.heartbeat.HeartbeatStatus

data class DashboardUiState(
    val deviceId: String? = null,
    val merchantId: String? = null,
    val queueDepth: Int = 0,
    val lastSyncAt: Long = 0,
    val heartbeat: HeartbeatOutcome? = null,
    val simMapping: Map<Int, SimProfile> = emptyMap(),
    val update: UpdateDecision? = null,
)

@HiltViewModel
class DashboardViewModel @Inject constructor(
    sessionStore: SessionStore,
    prefsStore: PrefsStore,
    outboxRepository: OutboxRepository,
    heartbeatStatus: HeartbeatStatus,
    private val configRepository: ConfigRepository,
) : ViewModel() {
    private val update = MutableStateFlow<UpdateDecision?>(null)

    val uiState: StateFlow<DashboardUiState> = combine(
        sessionStore.state,
        prefsStore.prefs,
        outboxRepository.pendingCount(),
        heartbeatStatus.state,
        update,
    ) { session, prefs, depth, heartbeat, update ->
        DashboardUiState(
            deviceId = session.deviceId,
            merchantId = session.merchantId,
            queueDepth = depth,
            lastSyncAt = prefs.lastSyncAt,
            heartbeat = heartbeat,
            simMapping = prefs.simMapping,
            update = update,
        )
    }.stateIn(
        scope = viewModelScope,
        started = SharingStarted.WhileSubscribed(5_000),
        initialValue = DashboardUiState(),
    )

    init {
        checkForUpdates()
    }

    fun checkForUpdates() {
        viewModelScope.launch {
            when (val result = configRepository.sync()) {
                is ConfigSyncResult.Success -> update.value = result.decision
                else -> Unit // keep the previous decision; update is best-effort
            }
        }
    }
}
