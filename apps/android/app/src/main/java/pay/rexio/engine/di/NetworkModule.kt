package pay.rexio.engine.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import java.util.concurrent.TimeUnit
import javax.inject.Singleton
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import pay.rexio.engine.BuildConfig
import pay.rexio.engine.core.time.Clock
import pay.rexio.engine.core.time.SystemClock
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.remote.AuthSource
import pay.rexio.engine.data.remote.HmacAuthInterceptor
import pay.rexio.engine.data.security.SecretStorage

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        encodeDefaults = false
    }

    @Provides
    @Singleton
    fun provideClock(): Clock = SystemClock

    @Provides
    @Singleton
    fun provideAuthSource(
        sessionStore: SessionStore,
        secretStorage: SecretStorage,
    ): AuthSource = object : AuthSource {
        override fun deviceId(): String? = sessionStore.current.deviceId
        override fun deviceSecret(): ByteArray? = secretStorage.deviceSecret()
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(authInterceptor: HmacAuthInterceptor): OkHttpClient {
        val builder = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(12, TimeUnit.SECONDS) // under the server's 15s read timeout
            .writeTimeout(10, TimeUnit.SECONDS)
            .addInterceptor(authInterceptor)
        if (BuildConfig.DEBUG) {
            // BASIC logs method + URL + status only — never bodies, so no SMS
            // content ever reaches the log.
            builder.addInterceptor(
                HttpLoggingInterceptor().apply { level = HttpLoggingInterceptor.Level.BASIC },
            )
        }
        return builder.build()
    }
}
