package pay.rexio.engine.work

import android.content.Context
import androidx.work.ExistingWorkPolicy
import androidx.work.WorkManager
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/** Ensures the upload chain exists; KEEP means a live chain is left to run. */
@Singleton
class UploadScheduler @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    fun enqueueUpload() {
        WorkManager.getInstance(context).enqueueUniqueWork(
            SmsUploadWorker.NAME,
            ExistingWorkPolicy.KEEP,
            SmsUploadWorker.oneTimeRequest(),
        )
    }
}
