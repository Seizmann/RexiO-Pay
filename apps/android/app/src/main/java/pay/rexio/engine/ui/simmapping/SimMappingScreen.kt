package pay.rexio.engine.ui.simmapping

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel

@Composable
fun SimMappingScreen(
    onDone: () -> Unit,
    viewModel: SimMappingViewModel = hiltViewModel(),
) {
    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 24.dp)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(32.dp))
            Text("Which SIM is which?", style = MaterialTheme.typography.headlineMedium)
            Spacer(Modifier.height(8.dp))
            Text(
                "Name each SIM slot after the payment profile it receives SMS for. " +
                    "Every forwarded SMS is tagged with its SIM slot — this mapping " +
                    "just makes the dashboard readable.",
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(Modifier.height(24.dp))

            viewModel.drafts.forEach { draft ->
                SlotEditor(
                    draft = draft,
                    numberError = viewModel.numberError,
                    onLabelChange = { viewModel.updateLabel(draft.slot, it) },
                    onNumberChange = { viewModel.updateNumber(draft.slot, it) },
                )
                Spacer(Modifier.height(16.dp))
            }

            Spacer(Modifier.height(8.dp))
            Button(
                onClick = { viewModel.save(onDone) },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("Save and continue")
            }
            Spacer(Modifier.height(8.dp))
            OutlinedButton(
                onClick = onDone,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("Skip for now")
            }
            Spacer(Modifier.height(32.dp))
        }
    }
}

@Composable
private fun SlotEditor(
    draft: SlotDraft,
    numberError: String?,
    onLabelChange: (String) -> Unit,
    onNumberChange: (String) -> Unit,
) {
    Column(modifier = Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text("SIM ${draft.slot + 1}", style = MaterialTheme.typography.titleMedium)
        OutlinedTextField(
            value = draft.label,
            onValueChange = onLabelChange,
            label = { Text("Label (e.g. bKash Personal)") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = draft.number,
            onValueChange = onNumberChange,
            label = { Text("MFS number (optional)") },
            placeholder = { Text("01712345678") },
            isError = numberError != null,
            supportingText = numberError?.let { { Text(it) } },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}
