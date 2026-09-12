package pay.rexio.engine.di

import android.content.Context
import androidx.room.Room
import dagger.Binds
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton
import pay.rexio.engine.data.local.OutboxDao
import pay.rexio.engine.data.local.RexioDatabase
import pay.rexio.engine.data.prefs.PrefsStore
import pay.rexio.engine.data.prefs.SessionStore
import pay.rexio.engine.data.security.SecretStorage
import pay.rexio.engine.data.security.SecretStore

@Module
@InstallIn(SingletonComponent::class)
abstract class DatabaseModule {
    @Binds
    @Singleton
    abstract fun bindSecretStorage(impl: SecretStore): SecretStorage

    companion object {
        @Provides
        @Singleton
        fun provideDatabase(@ApplicationContext context: Context): RexioDatabase =
            Room.databaseBuilder(context, RexioDatabase::class.java, "rexio.db").build()

        @Provides
        fun provideOutboxDao(db: RexioDatabase): OutboxDao = db.outboxDao()

        @Provides
        @Singleton
        fun providePrefsStore(@ApplicationContext context: Context): PrefsStore =
            PrefsStore(context, "rexio_prefs")

        @Provides
        @Singleton
        fun provideSessionStore(@ApplicationContext context: Context): SessionStore = SessionStore(context)

        @Provides
        @Singleton
        fun provideSecretStore(@ApplicationContext context: Context): SecretStore = SecretStore(context)
    }
}
