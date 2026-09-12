package pay.rexio.engine.data.security

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import pay.rexio.engine.core.crypto.HmacSigner

/**
 * Keystore-backed storage for the device secret. The pairing response returns
 * the raw 32-byte secret as unpadded base64url; we store that string here and
 * decode it per request — the decoded bytes are never cached anywhere.
 */
class SecretStore(context: Context) : SecretStorage {

    private val prefs: SharedPreferences by lazy {
        val masterKey = MasterKey.Builder(context.applicationContext)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context.applicationContext,
            PREF_FILE,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    override fun deviceSecret(): ByteArray? =
        prefs.getString(KEY_DEVICE_SECRET, null)?.let(HmacSigner::decodeSecret)

    override fun saveDeviceSecret(base64UrlSecret: String) {
        prefs.edit().putString(KEY_DEVICE_SECRET, base64UrlSecret).apply()
    }

    override fun clear() {
        prefs.edit().remove(KEY_DEVICE_SECRET).apply()
    }

    private companion object {
        const val PREF_FILE = "rexio_secret"
        const val KEY_DEVICE_SECRET = "device_secret"
    }
}
