package pay.rexio.engine.data.remote

import kotlinx.serialization.json.Json
import retrofit2.HttpException

/**
 * Normalized form of the backend's uniform error shape:
 * {"error": {"code": "...", "message": "...", "param": "..."}}.
 */
class ApiException(
    val code: String,
    message: String,
    val httpStatus: Int,
    val param: String? = null,
) : Exception(message)

suspend fun HttpException.toApiException(json: Json): ApiException {
    val envelope = runCatching {
        response()?.errorBody()?.string()?.let {
            json.decodeFromString(ErrorEnvelopeDto.serializer(), it)
        }
    }.getOrNull()
    return ApiException(
        code = envelope?.error?.code ?: "http_${code()}",
        message = envelope?.error?.message ?: message(),
        httpStatus = code(),
        param = envelope?.error?.param,
    )
}

/** Maps HTTP failures to ApiException; IO-level failures propagate unchanged. */
suspend fun <T> apiCall(json: Json, block: suspend () -> T): T = try {
    block()
} catch (e: HttpException) {
    throw e.toApiException(json)
}
