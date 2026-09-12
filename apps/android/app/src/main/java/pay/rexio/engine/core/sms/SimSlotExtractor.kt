package pay.rexio.engine.core.sms

/**
 * Extracts the SIM slot an SMS arrived on from the receiver intent's extras.
 *
 * Key list mirrors the reference app's OEM-variation detection (phone, slot,
 * simId, simSlot, slot_id, simnum, slotId, slotIdx, and the framework's
 * SLOT_INDEX extra), plus a generic fallback for any key whose name contains
 * "slot"/"sim". Unlike the reference implementation, values are read
 * type-safely (Bundle values may be Int or String) and no exception paths are
 * reachable; the reference's getInt on a String value crashes.
 *
 * Slots are returned 0-based as the OS reports them; UI labels are +1.
 * Returns null when the OEM firmware provided no slot information.
 */
object SimSlotExtractor {
    private val KNOWN_KEYS = listOf(
        "phone",
        "slot",
        "simId",
        "simSlot",
        "slot_id",
        "simnum",
        "slotId",
        "slotIdx",
        "android.telephony.extra.SLOT_INDEX",
    )

    fun extract(extras: Map<String, Any?>): Int? {
        for (key in KNOWN_KEYS) {
            val slot = toIntOrNull(extras[key])
            if (slot != null && slot >= 0) return slot
        }
        for ((key, value) in extras) {
            val lower = key.lowercase()
            if (!lower.contains("slot") && !lower.contains("sim")) continue
            val slot = (value as? String)?.toIntOrNull() ?: toIntOrNull(value)
            if (slot != null && slot in 0..2) return slot
        }
        return null
    }

    private fun toIntOrNull(value: Any?): Int? = when (value) {
        is Int -> value
        is Long -> if (value in Int.MIN_VALUE..Int.MAX_VALUE) value.toInt() else null
        is String -> value.toIntOrNull()
        else -> null
    }
}
