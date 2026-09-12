package pay.rexio.engine.data.remote

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

interface DeviceApi {
    /** Unsigned — the pairing token is the only credential. */
    @POST("v1/device/pair")
    suspend fun pair(@Body request: PairRequestDto): PairResponseDto

    @POST("v1/device/sms")
    suspend fun sendSms(@Body request: SmsRequestDto): SmsResponseDto

    @POST("v1/device/heartbeat")
    suspend fun heartbeat(@Body request: HeartbeatRequestDto): HeartbeatResponseDto

    @GET("v1/device/config")
    suspend fun config(): ConfigResponseDto
}
