// Package phone provides phone number canonicalization for Bangladeshi MFS numbers.
// All numbers are stored and compared in canonical form: 01XXXXXXXXX (11 digits).
package phone

import (
	"regexp"
	"strings"
)

var nonDigit = regexp.MustCompile(`\D`)

// Canonicalize strips non-digits, then removes +88/880/88 country-code prefixes,
// and returns 01XXXXXXXXX. Returns "" if the result is not a valid BD mobile number.
func Canonicalize(s string) string {
	// 1. Remove all non-digit characters (+, spaces, dashes, etc.)
	digits := nonDigit.ReplaceAllString(strings.TrimSpace(s), "")

	// 2. Strip country code prefixes (longest match first)
	switch {
	case strings.HasPrefix(digits, "8801"):
		digits = "0" + digits[3:] // 8801XXXXXXXXX → 01XXXXXXXXX
	case strings.HasPrefix(digits, "880"):
		digits = digits[3:] // bare 880 prefix without leading zero — unusual, handle anyway
	}

	// 3. Must be exactly 11 digits starting with 01
	if len(digits) == 11 && strings.HasPrefix(digits, "01") {
		return digits
	}
	return ""
}

// IsValid reports whether s is a valid Bangladeshi mobile number after canonicalization.
func IsValid(s string) bool {
	return Canonicalize(s) != ""
}
