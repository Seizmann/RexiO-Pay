package pay.rexio.engine.core.sim

import android.content.Context
import android.telephony.SubscriptionManager
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/** Device info sent with the pairing request. */
interface DeviceInfo {
    val model: String
    val androidVersion: String
    val appVersion: String
}

class AndroidDeviceInfo : DeviceInfo {
    override val model: String = android.os.Build.MODEL
    override val androidVersion: String = android.os.Build.VERSION.RELEASE
    override val appVersion: String = pay.rexio.engine.BuildConfig.VERSION_NAME
}

/**
 * Enumerates physical SIM slots for the SIM → profile mapping UI. Subscription
 * data needs READ_PHONE_STATE on newer Android, which this app deliberately
 * does not request — when the subscription list is unavailable we fall back to
 * two slots, the common dual-SIM setup for MFS phone numbers in Bangladesh.
 */
@Singleton
class SimSlotsProvider @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    fun slots(): List<Int> {
        // READ_PHONE_STATE is deliberately not requested; accessing the subscription
        // list will throw a SecurityException when the permission is absent, so the
        // whole call is inside runCatching and falls back to [0, 1]. Lint correctly
        // reports the call as missing-permission; we suppress because the fallback
        // is the design intent.
        @Suppress("MissingPermission")
        val subscriptionSlots = runCatching {
            context.getSystemService(SubscriptionManager::class.java)
                .activeSubscriptionInfoList
                ?.map { it.simSlotIndex }
                ?.filter { it >= 0 }
                ?.distinct()
                ?.sorted()
        }.getOrNull()
        return when {
            subscriptionSlots.isNullOrEmpty() -> listOf(0, 1)
            else -> subscriptionSlots
        }
    }

    fun label(slot: Int): String = "SIM ${slot + 1}"
}
