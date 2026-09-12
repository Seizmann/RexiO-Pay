package pay.rexio.engine.sms

import android.os.Bundle
import com.google.common.truth.Truth.assertThat
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class SmsReceiverTest {
    private val receiver = SmsReceiver()

    @Test
    fun `parse returns null without extras`() {
        assertThat(receiver.parse(null)).isNull()
    }

    @Test
    fun `parse returns null when pdus is missing`() {
        val bundle = Bundle().apply { putString("format", "3gpp") }
        assertThat(receiver.parse(bundle)).isNull()
    }

    @Test
    fun `parse returns null for an empty pdu array`() {
        val bundle = Bundle().apply { putParcelableArray("pdus", emptyArray()) }
        assertThat(receiver.parse(bundle)).isNull()
    }

    @Test
    fun `parse returns null for a malformed pdu array`() {
        val bundle = Bundle().apply {
            putSerializable("pdus", arrayOf("not", "byte", "arrays"))
        }
        assertThat(receiver.parse(bundle)).isNull()
    }
}
