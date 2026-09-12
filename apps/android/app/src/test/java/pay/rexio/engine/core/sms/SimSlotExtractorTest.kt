package pay.rexio.engine.core.sms

import com.google.common.truth.Truth.assertThat
import com.google.common.truth.Truth.assertWithMessage
import org.junit.Test

class SimSlotExtractorTest {
    @Test
    fun `reads every known OEM int key`() {
        val keys = listOf("phone", "slot", "simId", "simSlot", "slot_id", "simnum", "slotId", "slotIdx")
        keys.forEach { key ->
            assertWithMessage("key $key")
                .that(SimSlotExtractor.extract(mapOf("pdus" to Any(), key to 1)))
                .isEqualTo(1)
        }
    }

    @Test
    fun `reads the framework slot index extra`() {
        assertThat(
            SimSlotExtractor.extract(mapOf("android.telephony.extra.SLOT_INDEX" to 0)),
        ).isEqualTo(0)
    }

    @Test
    fun `reads known key stored as string without crashing`() {
        // The reference app calls Bundle.getInt here and throws on OEMs that
        // store the slot as a string; we read it type-safely instead.
        assertThat(SimSlotExtractor.extract(mapOf("slot" to "1"))).isEqualTo(1)
        assertThat(SimSlotExtractor.extract(mapOf("phone" to 1L))).isEqualTo(1)
    }

    @Test
    fun `generic fallback accepts slot or sim keys with values 0 through 2`() {
        assertThat(SimSlotExtractor.extract(mapOf("simIndex" to "2"))).isEqualTo(2)
        assertThat(SimSlotExtractor.extract(mapOf("oem_slot_id" to 1))).isEqualTo(1)
        assertThat(SimSlotExtractor.extract(mapOf("somethingSlotElse" to "0"))).isEqualTo(0)
    }

    @Test
    fun `generic fallback rejects out of range and non numeric values`() {
        assertThat(SimSlotExtractor.extract(mapOf("simIndex" to "3"))).isNull()
        assertThat(SimSlotExtractor.extract(mapOf("slotX" to "one"))).isNull()
    }

    @Test
    fun `returns null when no slot info is present`() {
        assertThat(SimSlotExtractor.extract(mapOf("pdus" to Any(), "format" to "3gpp"))).isNull()
        assertThat(SimSlotExtractor.extract(emptyMap())).isNull()
    }
}
