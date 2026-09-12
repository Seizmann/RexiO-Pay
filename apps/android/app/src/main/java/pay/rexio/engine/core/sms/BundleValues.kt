package pay.rexio.engine.core.sms

import android.os.Bundle

/** Bundle → value map for type-safe SIM-slot extraction. */
fun Bundle.toValueMap(): Map<String, Any?> = keySet().associateWith { get(it) }
