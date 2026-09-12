package pay.rexio.engine.core.pairing

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class QrPayload(
    @SerialName("server_url") val serverUrl: String,
    @SerialName("merchant_id") val merchantId: String,
    @SerialName("pairing_token") val pairingToken: String,
)

sealed interface QrParseResult {
    data class Success(val payload: QrPayload) : QrParseResult
    data class Failure(val reason: String) : QrParseResult
}

/**
 * Parses the pairing QR. Canonical content (also what the dashboard must
 * generate) is JSON {"server_url","merchant_id","pairing_token"}; a
 * rexio-pay://pair?... URI form is accepted as a fallback.
 */
object QrPayloadParser {
    private val json = Json { ignoreUnknownKeys = true }

    fun parse(raw: String): QrParseResult {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return QrParseResult.Failure("QR code is empty")
        parseJson(trimmed)?.let { return validate(it) }
        parseUri(trimmed)?.let { return validate(it) }
        return QrParseResult.Failure("Not a RexiO Pay pairing code")
    }

    private fun parseJson(raw: String): QrPayload? {
        if (!raw.startsWith("{")) return null
        return runCatching { json.decodeFromString(QrPayload.serializer(), raw) }.getOrNull()
    }

    private fun parseUri(raw: String): QrPayload? {
        if (!raw.startsWith("rexio-pay://pair")) return null
        val query = runCatching { java.net.URI(raw).rawQuery }.getOrNull() ?: return null
        fun param(name: String): String? = query
            .split('&')
            .mapNotNull { part ->
                val pieces = part.split('=', limit = 2)
                if (pieces.size == 2 && pieces[0] == name) {
                    runCatching { java.net.URLDecoder.decode(pieces[1], Charsets.UTF_8.name()) }.getOrNull()
                } else {
                    null
                }
            }
            .firstOrNull()
        return QrPayload(
            serverUrl = param("server_url") ?: return null,
            merchantId = param("merchant_id") ?: return null,
            pairingToken = param("pairing_token") ?: return null,
        )
    }

    private fun validate(payload: QrPayload): QrParseResult {
        if (!payload.serverUrl.startsWith("https://")) {
            return QrParseResult.Failure("server_url must use https")
        }
        if (payload.merchantId.isBlank()) return QrParseResult.Failure("merchant_id is missing")
        if (payload.pairingToken.isBlank()) return QrParseResult.Failure("pairing_token is missing")
        return QrParseResult.Success(payload)
    }
}
