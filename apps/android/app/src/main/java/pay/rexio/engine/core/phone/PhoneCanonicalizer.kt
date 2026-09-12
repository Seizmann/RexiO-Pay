package pay.rexio.engine.core.phone

/**
 * Port of apps/backend/internal/phone/phone.go. All numbers are stored and
 * compared in canonical form: 01XXXXXXXXX (11 digits). Returns "" when the
 * input is not a valid Bangladeshi mobile number.
 */
object PhoneCanonicalizer {
    private val nonDigit = Regex("\\D")

    fun canonicalize(input: String): String {
        var digits = nonDigit.replace(input.trim(), "")

        if (digits.startsWith("8801")) {
            digits = "0" + digits.substring(3) // 8801XXXXXXXXX → 01XXXXXXXXX
        } else if (digits.startsWith("880")) {
            digits = digits.substring(3) // bare 880 prefix without leading zero
        }

        return if (digits.length == 11 && digits.startsWith("01")) digits else ""
    }

    fun isValid(input: String): Boolean = canonicalize(input).isNotEmpty()
}
