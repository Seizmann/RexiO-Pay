package webhooks

import "time"

// RetrySchedule is the delay before attempts 1 through 5. A delivery is
// marked failed after the fifth unsuccessful attempt.
var RetrySchedule = [...]time.Duration{
	time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

const (
	DeliveryPending   = "pending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
)

// RetryDelay returns the delay after a failed attempt. attempt is one-based.
// Attempts beyond the configured schedule do not retry.
func RetryDelay(attempt int32) (time.Duration, bool) {
	if attempt < 1 || int(attempt) > len(RetrySchedule) {
		return 0, false
	}
	return RetrySchedule[attempt-1], true
}

// NextRetry returns the status and next retry time after attemptCount failed
// attempts. The returned time is based on now, making it deterministic for
// callers and tests. The status becomes failed after the last scheduled retry.
func NextRetry(now time.Time, attemptCount int32) (time.Time, string) {
	delay, ok := RetryDelay(attemptCount)
	if !ok {
		return now, DeliveryFailed
	}
	return now.Add(delay), DeliveryPending
}

// RetryStatus reports whether another attempt is scheduled after a failure.
func RetryStatus(attemptCount int32) string {
	if _, ok := RetryDelay(attemptCount); ok {
		return DeliveryPending
	}
	return DeliveryFailed
}
