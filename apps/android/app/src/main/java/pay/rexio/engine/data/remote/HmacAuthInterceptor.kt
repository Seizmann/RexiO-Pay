package pay.rexio.engine.data.remote

import javax.inject.Inject
import okhttp3.Interceptor
import okhttp3.Response
import okio.Buffer
import pay.rexio.engine.core.crypto.HmacSigner
import pay.rexio.engine.core.time.Clock

/** Supplies device identity to the signing interceptor; read per request. */
interface AuthSource {
    fun deviceId(): String?
    fun deviceSecret(): ByteArray?
}

/**
 * Adds X-Rexio-Device-Id / X-Rexio-Timestamp / X-Rexio-Signature per
 * REQUIREMENT §12.2. The signature is computed over the exact request body
 * bytes as sent, so this must stay an application interceptor (it re-reads
 * the body before OkHttp writes it to the wire).
 */
class HmacAuthInterceptor @Inject constructor(
    private val authSource: AuthSource,
    private val clock: Clock,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val deviceId = authSource.deviceId()
        val secret = authSource.deviceSecret()
        if (deviceId == null || secret == null) {
            // Pairing (or an unpaired device) goes through unsigned.
            return chain.proceed(request)
        }
        val rawBody = request.body?.let { body ->
            val buffer = Buffer()
            body.writeTo(buffer)
            buffer.readUtf8()
        } ?: ""
        val timestamp = clock.nowSeconds()
        val signature = HmacSigner.sign(secret, timestamp, rawBody)
        val signed = request.newBuilder()
            .header(HEADER_DEVICE_ID, deviceId)
            .header(HEADER_TIMESTAMP, timestamp.toString())
            .header(HEADER_SIGNATURE, signature)
            .build()
        return chain.proceed(signed)
    }

    companion object {
        const val HEADER_DEVICE_ID = "X-Rexio-Device-Id"
        const val HEADER_TIMESTAMP = "X-Rexio-Timestamp"
        const val HEADER_SIGNATURE = "X-Rexio-Signature"
    }
}
