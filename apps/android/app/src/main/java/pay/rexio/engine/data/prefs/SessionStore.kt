package pay.rexio.engine.data.prefs

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

data class SessionState(
    val deviceId: String? = null,
    val merchantId: String? = null,
) {
    val isPaired: Boolean get() = deviceId != null
}

/**
 * Synchronous session identity. Device ids are not secrets, so this lives in
 * plain SharedPreferences — the HMAC interceptor reads it without a coroutine
 * hop. The device secret itself is in SecretStore (Keystore-backed).
 */
class SessionStore(context: Context) {
    private val prefs = context.applicationContext
        .getSharedPreferences(PREF_FILE, Context.MODE_PRIVATE)

    private val _state = MutableStateFlow(load())
    val state: StateFlow<SessionState> = _state.asStateFlow()
    val current: SessionState get() = _state.value

    fun setPaired(deviceId: String, merchantId: String?) {
        prefs.edit().putString(KEY_DEVICE_ID, deviceId).apply()
        if (merchantId != null) {
            prefs.edit().putString(KEY_MERCHANT_ID, merchantId).apply()
        } else {
            prefs.edit().remove(KEY_MERCHANT_ID).apply()
        }
        _state.value = load()
    }

    fun clear() {
        prefs.edit().clear().apply()
        _state.value = load()
    }

    private fun load() = SessionState(
        deviceId = prefs.getString(KEY_DEVICE_ID, null),
        merchantId = prefs.getString(KEY_MERCHANT_ID, null),
    )

    private companion object {
        const val PREF_FILE = "rexio_session"
        const val KEY_DEVICE_ID = "device_id"
        const val KEY_MERCHANT_ID = "merchant_id"
    }
}
