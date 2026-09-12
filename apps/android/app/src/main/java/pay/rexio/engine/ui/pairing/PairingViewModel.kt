package pay.rexio.engine.ui.pairing

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.launch
import pay.rexio.engine.core.pairing.QrPayload
import pay.rexio.engine.core.pairing.QrParseResult
import pay.rexio.engine.core.pairing.QrPayloadParser
import pay.rexio.engine.domain.pairing.PairResult
import pay.rexio.engine.domain.pairing.PairingRepository

sealed interface PairingUiState {
    data object Scanning : PairingUiState
    data class Pairing(val serverUrl: String) : PairingUiState
    data class Success(val deviceId: String) : PairingUiState
    data class Error(val message: String) : PairingUiState
}

@HiltViewModel
class PairingViewModel @Inject constructor(
    private val pairingRepository: PairingRepository,
) : ViewModel() {
    var state by mutableStateOf<PairingUiState>(PairingUiState.Scanning)
        private set

    fun onQrScanned(raw: String) {
        if (state !is PairingUiState.Scanning) return
        when (val parsed = QrPayloadParser.parse(raw)) {
            is QrParseResult.Failure -> state = PairingUiState.Error(parsed.reason)
            is QrParseResult.Success -> pair(parsed.payload)
        }
    }

    fun resumeScanning() {
        state = PairingUiState.Scanning
    }

    private fun pair(payload: QrPayload) {
        state = PairingUiState.Pairing(payload.serverUrl)
        viewModelScope.launch {
            when (val result = pairingRepository.pair(payload)) {
                is PairResult.Success -> state = PairingUiState.Success(result.deviceId)
                is PairResult.Error -> state = PairingUiState.Error(result.message)
            }
        }
    }
}
