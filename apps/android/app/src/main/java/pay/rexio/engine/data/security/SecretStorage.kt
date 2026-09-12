package pay.rexio.engine.data.security

/**
 * Storage contract for the pairing device secret. The only production
 * implementation is Keystore-backed; tests swap in an in-memory fake (the
 * Android Keystore itself does not exist on the JVM).
 */
interface SecretStorage {
    fun deviceSecret(): ByteArray?
    fun saveDeviceSecret(base64UrlSecret: String)
    fun clear()
}
