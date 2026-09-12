package pay.rexio.engine.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val BrandIndigo = Color(0xFF533AFD)
private val BrandDark = Color(0xFF1C1E54)

private val RexioColorScheme = lightColorScheme(
    primary = BrandIndigo,
    onPrimary = Color.White,
    primaryContainer = Color(0xFFE4DFFF),
    onPrimaryContainer = BrandDark,
    secondary = BrandDark,
    background = Color(0xFFFAFAFC),
    surface = Color.White,
)

@Composable
fun RexioTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = RexioColorScheme,
        content = content,
    )
}
