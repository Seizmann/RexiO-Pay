package pay.rexio.engine.data.remote

import java.util.concurrent.ConcurrentHashMap
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory

/**
 * Builds DeviceApi instances per base URL (the server URL is user-overridable
 * in Settings, so a single cached instance per URL is the right shape).
 */
@Singleton
class DeviceApiProvider @Inject constructor(
    private val client: OkHttpClient,
    private val json: Json,
) {
    private val cache = ConcurrentHashMap<String, DeviceApi>()

    fun api(baseUrl: String): DeviceApi = cache.getOrPut(baseUrl) {
        val normalized = if (baseUrl.endsWith("/")) baseUrl else "$baseUrl/"
        Retrofit.Builder()
            .baseUrl(normalized)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(DeviceApi::class.java)
    }
}
