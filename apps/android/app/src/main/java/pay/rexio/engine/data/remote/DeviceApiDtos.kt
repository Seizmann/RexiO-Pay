package pay.rexio.engine.data.remote

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// DTOs mirror the backend's devices package field tags exactly
// (apps/backend/internal/devices/types.go). account_type is deliberately
// absent: the server rejects ingest when it contradicts the parsed message.

@Serializable
data class PairRequestDto(
    @SerialName("pairing_token") val pairingToken: String,
    @SerialName("device_name") val deviceName: String? = null,
    @SerialName("model") val model: String? = null,
    @SerialName("android_version") val androidVersion: String? = null,
    @SerialName("app_version") val appVersion: String? = null,
)

@Serializable
data class PairResponseDto(
    @SerialName("device_id") val deviceId: String,
    @SerialName("device_secret") val deviceSecret: String,
    @SerialName("status") val status: String,
)

@Serializable
data class SmsItemDto(
    @SerialName("client_id") val clientId: String,
    @SerialName("sender") val sender: String,
    @SerialName("body") val body: String,
    @SerialName("sms_time") val smsTime: String? = null,
    @SerialName("sim_slot") val simSlot: Int? = null,
    @SerialName("provider") val provider: String? = null,
)

@Serializable
data class SmsRequestDto(
    @SerialName("messages") val messages: List<SmsItemDto>,
)

@Serializable
data class SmsAckDto(
    @SerialName("client_id") val clientId: String,
    @SerialName("sms_id") val smsId: String? = null,
    @SerialName("status") val status: String,
)

@Serializable
data class SmsResponseDto(
    @SerialName("acks") val acks: List<SmsAckDto>,
)

@Serializable
data class HeartbeatRequestDto(
    @SerialName("battery_level") val batteryLevel: Int,
    @SerialName("app_version") val appVersion: String,
    @SerialName("queue_depth") val queueDepth: Int,
)

@Serializable
data class HeartbeatResponseDto(
    @SerialName("status") val status: String,
)

@Serializable
data class ConfigResponseDto(
    @SerialName("latest_app_version") val latestAppVersion: String? = null,
    @SerialName("apk_url") val apkUrl: String? = null,
    @SerialName("mandatory_update") val mandatoryUpdate: Boolean = false,
    @SerialName("parser_version") val parserVersion: String? = null,
    @SerialName("sender_ids") val senderIds: Map<String, List<String>> = emptyMap(),
)

@Serializable
data class ErrorEnvelopeDto(
    val error: ErrorBodyDto,
)

@Serializable
data class ErrorBodyDto(
    val code: String,
    val message: String,
    val param: String? = null,
)
