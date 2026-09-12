package pay.rexio.engine.data.prefs

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.google.common.truth.Truth.assertThat
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PrefsStoreTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

    private fun newStore() = PrefsStore(context, "test_prefs_${System.nanoTime()}")

    @Test
    fun `defaults to the production server url`() = runTest {
        val prefs = newStore().prefs.first()
        assertThat(prefs.serverUrl).isEqualTo("https://api.pay.rexio.pro")
        assertThat(prefs.senderIds).isEmpty()
        assertThat(prefs.inboxScanCursor).isEqualTo(0)
        assertThat(prefs.lastSyncAt).isEqualTo(0)
    }

    @Test
    fun `server url override strips trailing slash`() = runTest {
        val store = newStore()
        store.setServerUrl("http://10.0.2.2:8080/")
        assertThat(store.prefs.first().serverUrl).isEqualTo("http://10.0.2.2:8080")
    }

    @Test
    fun `sender ids cache round trip`() = runTest {
        val store = newStore()
        store.updateSenderIds(mapOf("bkash" to listOf("bKash", "16247")))
        assertThat(store.prefs.first().senderIds["bkash"]).containsExactly("bKash", "16247")
    }

    @Test
    fun `cursors and auth cooldown persist`() = runTest {
        val store = newStore()
        store.setInboxScanCursor(1_000L)
        store.setLastSyncAt(2_000L)
        store.setAuthCooldownUntil(3_000L)
        val prefs = store.prefs.first()
        assertThat(prefs.inboxScanCursor).isEqualTo(1_000L)
        assertThat(prefs.lastSyncAt).isEqualTo(2_000L)
        assertThat(prefs.authCooldownUntil).isEqualTo(3_000L)
    }

    @Test
    fun `resetState clears per-pairing data but keeps the server url`() = runTest {
        val store = newStore()
        store.setServerUrl("http://10.0.2.2:8080")
        store.updateSenderIds(mapOf("bkash" to listOf("bKash")))
        store.setInboxScanCursor(5_000L)
        store.setAuthCooldownUntil(6_000L)
        store.resetState()
        val prefs = store.prefs.first()
        assertThat(prefs.serverUrl).isEqualTo("http://10.0.2.2:8080")
        assertThat(prefs.senderIds).isEmpty()
        assertThat(prefs.inboxScanCursor).isEqualTo(0)
        assertThat(prefs.authCooldownUntil).isEqualTo(0)
    }
}
