package pay.rexio.engine.ui.settings

import android.content.Intent
import android.net.Uri
import android.os.PowerManager
import android.provider.Settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import pay.rexio.engine.service.HeartbeatService

@Composable
fun SettingsScreen(
    onBack: () -> Unit,
    onUnpaired: () -> Unit,
    viewModel: SettingsViewModel = hiltViewModel(),
) {
    val context = LocalContext.current
    val serverUrl by viewModel.serverUrl.collectAsState()
    var urlInput by remember { mutableStateOf("") }
    var urlSaved by remember { mutableStateOf(false) }
    var confirmUnpair by remember { mutableStateOf(false) }

    LaunchedEffect(serverUrl) {
        urlInput = serverUrl
    }

    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            OutlinedButton(onClick = onBack) { Text("Back") }

            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Server", style = MaterialTheme.typography.titleMedium)
                    OutlinedTextField(
                        value = urlInput,
                        onValueChange = {
                            urlInput = it
                            urlSaved = false
                        },
                        label = { Text("Server URL") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    if (urlSaved) {
                        Text(
                            "Saved. The new URL applies to the next pairing.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                    Text(
                        "Changing the server URL invalidates this device's pairing — " +
                            "pair again with a QR from the new server.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline,
                    )
                    Button(
                        onClick = {
                            viewModel.saveServerUrl(urlInput) { urlSaved = true }
                        },
                    ) {
                        Text("Save URL")
                    }
                }
            }

            BatteryCard(context)

            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("App update", style = MaterialTheme.typography.titleMedium)
                    OutlinedButton(onClick = viewModel::checkForUpdates) { Text("Check for update") }
                    viewModel.updateResult?.let { Text(it) }
                }
            }

            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Danger zone", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Unpairing stops SMS forwarding. You will need a new pairing " +
                            "QR from the dashboard to reconnect.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline,
                    )
                    Button(onClick = { confirmUnpair = true }) { Text("Unpair device") }
                }
            }
            Spacer(Modifier.height(24.dp))
        }
    }

    if (confirmUnpair) {
        AlertDialog(
            onDismissRequest = { confirmUnpair = false },
            title = { Text("Unpair this device?") },
            text = { Text("The device secret is deleted and SMS forwarding stops until you pair again.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        confirmUnpair = false
                        HeartbeatService.stop(context)
                        viewModel.unpair(onDone = onUnpaired)
                    },
                ) {
                    Text("Unpair", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmUnpair = false }) { Text("Cancel") }
            },
        )
    }
}

@Composable
private fun BatteryCard(context: android.content.Context) {
    val pm = remember { context.getSystemService(PowerManager::class.java) }
    var ignoring by remember {
        mutableStateOf(pm.isIgnoringBatteryOptimizations(context.packageName))
    }
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                ignoring = pm.isIgnoringBatteryOptimizations(context.packageName)
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Battery optimization", style = MaterialTheme.typography.titleMedium)
            if (ignoring) {
                Text(
                    "Battery optimization is off — SMS forwarding stays alive.",
                    color = MaterialTheme.colorScheme.primary,
                )
            } else {
                Text(
                    "Your phone may kill the SMS forwarder in the background. On Xiaomi, " +
                        "Oppo and Realme devices also enable Auto-start in the phone's " +
                        "app settings for this app.",
                    style = MaterialTheme.typography.bodySmall,
                )
                OutlinedButton(
                    onClick = {
                        val intent = Intent(
                            Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                            Uri.parse("package:${context.packageName}"),
                        )
                        context.startActivity(intent)
                    },
                ) {
                    Text("Turn off battery optimization")
                }
            }
        }
    }
}
