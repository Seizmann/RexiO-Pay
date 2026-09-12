package webhooks

import (
	"testing"
	"time"
)

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int32
		want    time.Duration
		ok      bool
	}{
		{attempt: 0, ok: false},
		{attempt: 1, want: time.Minute, ok: true},
		{attempt: 2, want: 5 * time.Minute, ok: true},
		{attempt: 3, want: 30 * time.Minute, ok: true},
		{attempt: 4, want: 2 * time.Hour, ok: true},
		{attempt: 5, want: 12 * time.Hour, ok: true},
		{attempt: 6, ok: false},
	}
	for _, tt := range tests {
		got, ok := RetryDelay(tt.attempt)
		if got != tt.want || ok != tt.ok {
			t.Errorf("RetryDelay(%d) = (%s, %t), want (%s, %t)", tt.attempt, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNextRetry(t *testing.T) {
	now := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		attempt int32
		want    time.Time
		status  string
	}{
		{attempt: 1, want: now.Add(time.Minute), status: DeliveryPending},
		{attempt: 5, want: now.Add(12 * time.Hour), status: DeliveryPending},
		{attempt: 6, want: now, status: DeliveryFailed},
	} {
		got, status := NextRetry(now, tt.attempt)
		if !got.Equal(tt.want) || status != tt.status {
			t.Errorf("NextRetry(%d) = (%s, %q), want (%s, %q)", tt.attempt, got, status, tt.want, tt.status)
		}
	}
}
