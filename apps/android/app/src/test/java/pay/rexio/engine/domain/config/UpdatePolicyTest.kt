package pay.rexio.engine.domain.config

import com.google.common.truth.Truth.assertThat
import org.junit.Test

class UpdatePolicyTest {
    @Test
    fun `no latest version means no update`() {
        val decision = UpdatePolicy.evaluate("1.0.0", null, mandatoryUpdate = true, apkUrl = null)
        assertThat(decision.updateAvailable).isFalse()
        assertThat(decision.mandatory).isFalse()
    }

    @Test
    fun `same version means no update`() {
        val decision = UpdatePolicy.evaluate("1.0.0", "1.0.0", mandatoryUpdate = true, apkUrl = null)
        assertThat(decision.updateAvailable).isFalse()
    }

    @Test
    fun `newer version with mandatory flag is a blocking update`() {
        val decision = UpdatePolicy.evaluate("1.0.0", "1.1.0", mandatoryUpdate = true, apkUrl = "https://x/a.apk")
        assertThat(decision.updateAvailable).isTrue()
        assertThat(decision.mandatory).isTrue()
        assertThat(decision.apkUrl).isEqualTo("https://x/a.apk")
    }

    @Test
    fun `newer version without mandatory flag is optional`() {
        val decision = UpdatePolicy.evaluate("1.0.0", "1.1.0", mandatoryUpdate = false, apkUrl = null)
        assertThat(decision.updateAvailable).isTrue()
        assertThat(decision.mandatory).isFalse()
    }
}
