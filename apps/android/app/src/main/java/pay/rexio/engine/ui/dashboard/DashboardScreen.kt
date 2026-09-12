package pay.rexio.engine.ui.dashboard

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.hilt.navigation.compose.hiltViewModel
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import pay.rexio.engine.domain.heartbeat.HeartbeatOutcome

@Composable
fun DashboardScreen(
    onSettings: () -> Unit,
    onRePair: () -> Unit,
    viewModel: DashboardViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsState()
    val context = LocalContext.current

    val notificationPermissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { }

    LaunchedEffect(Unit) {
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }

    val update = state.update
    if (update != null && update.mandatory) {
        MandatoryUpdateOverlay(
            latestVersion = update.latestVersion ?: "",
            apkUrl = update.apkUrl,
        )
        return
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
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column {
                    Text("RexiO Pay Engine", style = MaterialTheme.typography.titleLarge)
                    Text(
                        "Device ${state.deviceId ?: "unknown"}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                OutlinedButton(onClick = onSettings) { Text("Settings") }
            }

            StatusCard(state)

            if (update != null && update.updateAvailable) {
                UpdateCard(update = update)
            }

            SimMappingCard(state.simMapping)

            Card(modifier = Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Device", style = MaterialTheme.typography.titleMedium)
                    Text("Paired to merchant ${state.merchantId ?: "unknown"}")
                    OutlinedButton(onClick = onRePair) { Text("Pair a different device") }
                }
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
private fun StatusCard(state: DashboardUiState) {
    val (statusText, statusColor) = when (state.heartbeat) {
        null -> "Waiting for first heartbeat" to MaterialTheme.colorScheme.outline
        HeartbeatOutcome.Ok -> "Connected" to MaterialTheme.colorScheme.primary
        HeartbeatOutcome.AuthPaused ->
            "Signing paused — check the device clock" to MaterialTheme.colorScheme.error
        HeartbeatOutcome.Unpaired -> "Not paired" to MaterialTheme.colorScheme.error
        is HeartbeatOutcome.Failed -> "Offline" to MaterialTheme.colorScheme.error
    }
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("Status", style = MaterialTheme.typography.titleMedium)
            Text(statusText, color = statusColor, style = MaterialTheme.typography.bodyLarge)
            Text("Queue: ${state.queueDepth} SMS waiting")
            Text(
                if (state.lastSyncAt == 0L) {
                    "Last sync: never"
                } else {
                    "Last sync: ${SimpleDateFormat("dd MMM, HH:mm:ss", Locale.US).format(Date(state.lastSyncAt))}"
                },
            )
        }
    }
}

@Composable
private fun SimMappingCard(mapping: Map<Int, pay.rexio.engine.data.prefs.SimProfile>) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("SIM profiles", style = MaterialTheme.typography.titleMedium)
            if (mapping.isEmpty()) {
                Text("No SIM mapping set yet.")
            } else {
                mapping.entries.sortedBy { it.key }.forEach { (slot, profile) ->
                    Text(
                        "SIM ${slot + 1}: ${profile.label}" +
                            (profile.mfsNumber.takeIf { it.isNotBlank() }?.let { " — $it" } ?: ""),
                    )
                }
            }
        }
    }
}

@Composable
private fun UpdateCard(update: pay.rexio.engine.domain.config.UpdateDecision) {
    val context = LocalContext.current
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Update available", style = MaterialTheme.typography.titleMedium)
            Text("Version ${update.latestVersion} is ready to install.")
            if (update.apkUrl != null) {
                Button(
                    onClick = {
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(update.apkUrl)))
                    },
                ) {
                    Text("Download update")
                }
            } else {
                Text(
                    "Ask support for the latest APK link.",
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

@Composable
private fun MandatoryUpdateOverlay(latestVersion: String, apkUrl: String?) {
    val context = LocalContext.current
    Box(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Update required", style = MaterialTheme.typography.headlineSmall)
                Text(
                    "This app version can no longer forward payments. " +
                        "Install version $latestVersion to continue.",
                )
                if (apkUrl != null) {
                    Button(
                        onClick = {
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(apkUrl)))
                        },
                    ) {
                        Text("Download update")
                    }
                } else {
                    Text(
                        "Ask support for the latest APK link.",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
    }
}
