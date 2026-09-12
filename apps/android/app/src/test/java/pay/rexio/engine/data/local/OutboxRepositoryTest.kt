package pay.rexio.engine.data.local

import android.content.Context
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import com.google.common.truth.Truth.assertThat
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class OutboxRepositoryTest {
    private lateinit var db: RexioDatabase
    private lateinit var repository: OutboxRepository

    @Before
    fun setUp() {
        val context = ApplicationProvider.getApplicationContext<Context>()
        db = Room.inMemoryDatabaseBuilder(context, RexioDatabase::class.java)
            .allowMainThreadQueries()
            .build()
        repository = OutboxRepository(db.outboxDao())
    }

    @After
    fun tearDown() {
        db.close()
    }

    @Test
    fun `inserts an sms and exposes it as pending`() = runTest {
        val inserted = repository.enqueue("bKash", "You have received Tk 100", 0, RECEIVED_AT)
        assertThat(inserted).isTrue()
        val pending = repository.pendingBatch(now = RECEIVED_AT + 1)
        assertThat(pending).hasSize(1)
        assertThat(pending[0].sender).isEqualTo("bKash")
        assertThat(pending[0].simSlot).isEqualTo(0)
    }

    @Test
    fun `dedupes identical sms within the window`() = runTest {
        assertThat(repository.enqueue("bKash", "same body", 1, RECEIVED_AT)).isTrue()
        assertThat(repository.enqueue("bKash", "same body", 1, RECEIVED_AT + 30_000)).isFalse()
        assertThat(repository.enqueue(" bKash ", " same body ", 1, RECEIVED_AT + 59_999)).isFalse()
        assertThat(repository.pendingBatch(now = RECEIVED_AT + 60_000)).hasSize(1)
    }

    @Test
    fun `inserts again when the window has passed or sender differs`() = runTest {
        repository.enqueue("bKash", "same body", 0, RECEIVED_AT)
        assertThat(repository.enqueue("bKash", "same body", 0, RECEIVED_AT + 60_001)).isTrue()
        assertThat(repository.enqueue("NAGAD", "same body", 0, RECEIVED_AT + 1_000)).isTrue()
        assertThat(repository.pendingBatch(now = RECEIVED_AT + 70_000)).hasSize(3)
    }

    @Test
    fun `rejects blank bodies`() = runTest {
        assertThat(repository.enqueue("bKash", "   ", 0, RECEIVED_AT)).isFalse()
        assertThat(repository.pendingCountOnce()).isEqualTo(0)
    }

    @Test
    fun `pending batch is chronological oldest first`() = runTest {
        repository.enqueue("bKash", "older", null, RECEIVED_AT)
        repository.enqueue("NAGAD", "newer", null, RECEIVED_AT + 10_000)
        repository.enqueue("bKash", "newest", null, RECEIVED_AT + 20_000)
        val pending = repository.pendingBatch(now = RECEIVED_AT + 30_000)
        assertThat(pending.map { it.rawBody }).containsExactly("older", "newer", "newest").inOrder()
    }

    @Test
    fun `deletes acked rows and reports remaining queue depth`() = runTest {
        repository.enqueue("bKash", "a", null, RECEIVED_AT)
        repository.enqueue("bKash", "b", null, RECEIVED_AT + 1_000)
        val pending = repository.pendingBatch(now = RECEIVED_AT + 2_000)
        repository.deleteAcked(pending.map { it.id })
        assertThat(repository.pendingCountOnce()).isEqualTo(0)
        assertThat(repository.pendingCount().first()).isEqualTo(0)
    }

    @Test
    fun `skips rows in cooldown`() = runTest {
        repository.enqueue("bKash", "resting", null, RECEIVED_AT)
        val row = repository.pendingBatch(now = RECEIVED_AT + 1).single()
        val cooldownUntil = RECEIVED_AT + 6 * 60 * 60 * 1000
        repository.cooldown(listOf(row.id), cooldownUntil)

        assertThat(repository.pendingBatch(now = cooldownUntil - 1)).isEmpty()
        assertThat(repository.pendingBatch(now = cooldownUntil)).hasSize(1)
    }

    private companion object {
        const val RECEIVED_AT = 1_700_000_000_000L
    }
}
