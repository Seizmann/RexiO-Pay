package pay.rexio.engine.work

import com.google.common.truth.Truth.assertThat
import org.junit.Test

class UploadRetryPolicyTest {
    @Test
    fun `attempt cap trips only past ten attempts`() {
        assertThat(UploadRetryPolicy.exceeded(1)).isFalse()
        assertThat(UploadRetryPolicy.exceeded(10)).isFalse()
        assertThat(UploadRetryPolicy.exceeded(11)).isTrue()
    }
}
