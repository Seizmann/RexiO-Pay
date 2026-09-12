package pay.rexio.engine.ui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.service.HeartbeatService
import pay.rexio.engine.ui.dashboard.DashboardScreen
import pay.rexio.engine.ui.pairing.PairingScreen
import pay.rexio.engine.ui.settings.SettingsScreen
import pay.rexio.engine.ui.simmapping.SimMappingScreen
import pay.rexio.engine.ui.theme.RexioTheme

object Routes {
    const val PAIRING = "pairing"
    const val SIM_MAPPING = "sim_mapping"
    const val DASHBOARD = "dashboard"
    const val SETTINGS = "settings"
}

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    @Inject
    lateinit var sessionStore: SessionStore

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            RexioTheme {
                RexioNavHost(sessionStore)
            }
        }
    }
}

@Composable
fun RexioNavHost(sessionStore: SessionStore) {
    val navController = rememberNavController()
    val context = LocalContext.current
    val startDestination = remember {
        if (sessionStore.current.isPaired) Routes.DASHBOARD else Routes.PAIRING
    }
    LaunchedEffect(Unit) {
        if (sessionStore.current.isPaired) HeartbeatService.start(context)
    }
    NavHost(navController = navController, startDestination = startDestination) {
        composable(Routes.PAIRING) {
            PairingScreen(
                onPaired = {
                    navController.navigate(Routes.SIM_MAPPING) {
                        popUpTo(Routes.PAIRING) { inclusive = true }
                    }
                },
            )
        }
        composable(Routes.SIM_MAPPING) {
            SimMappingScreen(
                onDone = {
                    navController.navigate(Routes.DASHBOARD) {
                        popUpTo(Routes.SIM_MAPPING) { inclusive = true }
                    }
                },
            )
        }
        composable(Routes.DASHBOARD) {
            DashboardScreen(
                onSettings = { navController.navigate(Routes.SETTINGS) },
                onRePair = {
                    navController.navigate(Routes.PAIRING) {
                        popUpTo(Routes.DASHBOARD) { inclusive = true }
                    }
                },
            )
        }
        composable(Routes.SETTINGS) {
            SettingsScreen(
                onBack = { navController.popBackStack() },
                onUnpaired = {
                    navController.navigate(Routes.PAIRING) {
                        popUpTo(navController.graph.id) { inclusive = true }
                    }
                },
            )
        }
    }
}
