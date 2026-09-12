package devices

import (
	"testing"
	"time"
)

func TestPairingExpired(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		expires time.Time
		want    bool
	}{
		{name: "before expiry", expires: now.Add(time.Second), want: false},
		{name: "at expiry", expires: now, want: true},
		{name: "after expiry", expires: now.Add(-time.Second), want: true},
		{name: "zero expiry", expires: time.Time{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PairingExpired(tt.expires, now); got != tt.want {
				t.Fatalf("PairingExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
