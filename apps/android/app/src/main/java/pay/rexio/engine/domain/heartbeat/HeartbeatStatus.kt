package pay.rexio.engine.domain.heartbeat

import android.content.Context
import android.os.BatteryManager
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

fun interface BatteryLevelProvider {
    /** Battery percentage 0–100, or -1 when unknown. */
    fun current(): Int
}

class SystemBatteryLevelProvider @Inject constructor(
    @ApplicationContext private val context: Context,
) : BatteryLevelProvider {
    override fun current(): Int = runCatching {
        context.getSystemService(BatteryManager::class.java)
            .getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY)
    }.getOrDefault(-1)
}

/** Latest heartbeat result, surfaced on the dashboard. */
@Singleton
class HeartbeatStatus @Inject constructor() {
    private val _state = MutableStateFlow<HeartbeatOutcome?>(null)
    val state: StateFlow<HeartbeatOutcome?> = _state.asStateFlow()

    fun update(outcome: HeartbeatOutcome) {
        _state.value = outcome
    }
}
