package pay.rexio.engine.data.prefs

import kotlinx.serialization.Serializable

/** Local-only SIM slot → profile mapping; the backend has no device profiles API. */
@Serializable
data class SimProfile(
    val label: String,
    /** Canonical 01XXXXXXXXX MFS number, or "" when unset. */
    val mfsNumber: String = "",
)
