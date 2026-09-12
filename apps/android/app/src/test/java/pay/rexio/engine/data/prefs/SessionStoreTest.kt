package pay.rexio.engine.data.prefs

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.google.common.truth.Truth.assertThat
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class SessionStoreTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

    @Test
    fun `pairing state round trip`() {
        val store = SessionStore(context)
        assertThat(store.current.isPaired).isFalse()
        assertThat(store.current.deviceId).isNull()

        store.setPaired("dev_abc123", "mer_xyz")
        assertThat(store.current.isPaired).isTrue()
        assertThat(store.current.deviceId).isEqualTo("dev_abc123")
        assertThat(store.current.merchantId).isEqualTo("mer_xyz")

        store.clear()
        assertThat(store.current.isPaired).isFalse()
        assertThat(store.current.deviceId).isNull()
        assertThat(store.current.merchantId).isNull()
    }

    @Test
    fun `state flow reflects writes`() {
        val store = SessionStore(context)
        assertThat(store.state.value.isPaired).isFalse()
        store.setPaired("dev_1", null)
        assertThat(store.state.value.deviceId).isEqualTo("dev_1")
    }
}
