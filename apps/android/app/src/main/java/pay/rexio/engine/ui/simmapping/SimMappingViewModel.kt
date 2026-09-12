package pay.rexio.engine.ui.simmapping

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.launch
import pay.rexio.engine.core.phone.PhoneCanonicalizer
import pay.rexio.engine.core.sim.SimSlotsProvider
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SimProfile

data class SlotDraft(
    val slot: Int,
    val label: String,
    val number: String,
)

@HiltViewModel
class SimMappingViewModel @Inject constructor(
    private val simSlotsProvider: SimSlotsProvider,
    private val prefsStore: PrefsStore,
) : ViewModel() {
    var drafts by mutableStateOf<List<SlotDraft>>(emptyList())
        private set
    var numberError by mutableStateOf<String?>(null)
        private set

    init {
        drafts = simSlotsProvider.slots().map { slot ->
            SlotDraft(slot = slot, label = "SIM ${slot + 1}", number = "")
        }
    }

    fun updateLabel(slot: Int, label: String) {
        drafts = drafts.map { if (it.slot == slot) it.copy(label = label) else it }
    }

    fun updateNumber(slot: Int, number: String) {
        numberError = null
        drafts = drafts.map { if (it.slot == slot) it.copy(number = number) else it }
    }

    fun save(onSaved: () -> Unit) {
        val mapping = mutableMapOf<Int, SimProfile>()
        for (draft in drafts) {
            val canonical = if (draft.number.isBlank()) {
                ""
            } else {
                PhoneCanonicalizer.canonicalize(draft.number)
            }
            if (draft.number.isNotBlank() && canonical.isEmpty()) {
                numberError = "Enter a valid 11-digit MFS number like 01712345678"
                return
            }
            mapping[draft.slot] = SimProfile(
                label = draft.label.ifBlank { "SIM ${draft.slot + 1}" },
                mfsNumber = canonical,
            )
        }
        viewModelScope.launch {
            prefsStore.updateSimMapping(mapping)
            onSaved()
        }
    }
}
