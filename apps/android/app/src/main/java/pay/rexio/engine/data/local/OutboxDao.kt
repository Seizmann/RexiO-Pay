package pay.rexio.engine.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import kotlinx.coroutines.flow.Flow

@Dao
interface OutboxDao {
    @Query(
        "SELECT COUNT(*) FROM outbox WHERE sender = :sender AND raw_body = :body " +
            "AND received_at BETWEEN :windowStart AND :windowEnd",
    )
    suspend fun countDuplicates(sender: String, body: String, windowStart: Long, windowEnd: Long): Int

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun insert(row: OutboxEntity): Long

    /**
     * Oldest-first pending rows, skipping rows still in cooldown. The retry
     * budget (10 run attempts) is enforced by SmsUploadWorker, which puts
     * exhausted batches into cooldown instead of dropping them.
     */
    @Query(
        "SELECT * FROM outbox WHERE synced = 0 AND cooldown_until <= :now " +
            "ORDER BY received_at ASC, id ASC LIMIT :limit",
    )
    suspend fun pendingOldestFirst(limit: Int, now: Long): List<OutboxEntity>

    @Query("DELETE FROM outbox WHERE id IN (:ids)")
    suspend fun deleteByIds(ids: List<Long>)

    @Query("UPDATE outbox SET attempts = attempts + 1 WHERE id IN (:ids)")
    suspend fun incrementAttempts(ids: List<Long>)

    @Query("UPDATE outbox SET cooldown_until = :until WHERE id IN (:ids)")
    suspend fun setCooldown(ids: List<Long>, until: Long)

    @Query("SELECT COUNT(*) FROM outbox WHERE synced = 0")
    fun pendingCount(): Flow<Int>

    @Query("SELECT COUNT(*) FROM outbox WHERE synced = 0")
    suspend fun pendingCountOnce(): Int
}
