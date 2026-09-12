package pay.rexio.engine.domain.config

data class UpdateDecision(
    val updateAvailable: Boolean,
    val mandatory: Boolean,
    val latestVersion: String?,
    val apkUrl: String?,
)

/**
 * Pure update-decision logic: the device config reports the latest app version
 * plus a mandatory flag; we compare against the running versionName.
 */
object UpdatePolicy {
    fun evaluate(
        currentVersion: String,
        latestVersion: String?,
        mandatoryUpdate: Boolean,
        apkUrl: String?,
    ): UpdateDecision {
        val available = latestVersion != null &&
            latestVersion.isNotBlank() &&
            latestVersion != currentVersion
        return UpdateDecision(
            updateAvailable = available,
            mandatory = available && mandatoryUpdate,
            latestVersion = latestVersion,
            apkUrl = apkUrl,
        )
    }
}
