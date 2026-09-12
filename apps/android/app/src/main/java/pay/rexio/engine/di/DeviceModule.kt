package pay.rexio.engine.di

import android.content.Context
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton
import pay.rexio.engine.core.sim.AndroidDeviceInfo
import pay.rexio.engine.core.sim.DeviceInfo
import pay.rexio.engine.domain.heartbeat.BatteryLevelProvider
import pay.rexio.engine.domain.heartbeat.SystemBatteryLevelProvider

@Module
@InstallIn(SingletonComponent::class)
object DeviceModule {
    @Provides
    @Singleton
    fun provideDeviceInfo(): DeviceInfo = AndroidDeviceInfo()

    @Provides
    @Singleton
    fun provideBatteryLevelProvider(
        @ApplicationContext context: Context,
    ): BatteryLevelProvider = SystemBatteryLevelProvider(context)
}
