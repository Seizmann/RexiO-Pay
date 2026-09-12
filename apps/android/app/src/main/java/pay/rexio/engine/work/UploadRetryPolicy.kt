package pay.rexio.engine.work

/**
 * Retry budget for the upload chain: the worker gives up after this many run
 * attempts and puts the batch into cooldown (see OutboxRepository.COOLDOWN_MS),
 * after which a fresh heartbeat/config cycle picks the rows up again.
 */
object UploadRetryPolicy {
    const val MAX_RUN_ATTEMPTS = 10

    fun exceeded(runAttemptCount: Int): Boolean = runAttemptCount > MAX_RUN_ATTEMPTS
}
