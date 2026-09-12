package pay.rexio.engine.core.crypto

import java.util.Base64
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

/**
 * HMAC-SHA256 request signing, matching the backend's devices.VerifySignature
 * (REQUIREMENT §12.2): signature = hex(HMAC-SHA256(raw_device_secret, ts + "." + raw_body)).
 */
object HmacSigner {
    private const val ALGORITHM = "HmacSHA256"

    fun canonicalString(timestampSeconds: Long, rawBody: String): String =
        "$timestampSeconds.$rawBody"

    fun sign(secretBytes: ByteArray, timestampSeconds: Long, rawBody: String): String {
        val mac = Mac.getInstance(ALGORITHM)
        mac.init(SecretKeySpec(secretBytes, ALGORITHM))
        return mac.doFinal(canonicalString(timestampSeconds, rawBody).toByteArray(Charsets.UTF_8)).toHex()
    }

    /**
     * The pairing response returns the 32-byte device secret as unpadded base64url
     * (43 chars). Decode it once for use as the HMAC key.
     */
    fun decodeSecret(base64Url: String): ByteArray = Base64.getUrlDecoder().decode(base64Url)

    fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it) }
}
