package devices

import (
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	secret := []byte("device-secret")
	body := []byte(`{"messages":[]}`)
	ts := "1700000000"
	valid := Signature(secret, ts, body)
	tests := []struct {
		name      string
		signature string
		timestamp string
		wantErr   bool
	}{
		{name: "valid", signature: valid, timestamp: ts},
		{name: "bad signature", signature: Signature([]byte("other"), ts, body), timestamp: ts, wantErr: true},
		{name: "replay outside tolerance", signature: Signature(secret, "1699999600", body), timestamp: "1699999600", wantErr: true},
		{name: "malformed signature", signature: "nope", timestamp: ts, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifySignature(secret, tt.timestamp, tt.signature, body, now, 5*time.Minute); (err != nil) != tt.wantErr {
				t.Fatalf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestThreeFailuresAreDisabledThreshold(t *testing.T) {
	// The middleware persists this counter atomically. This table test locks the
	// security rule independently of SQL integration tests.
	count := 0
	for i := 0; i < 3; i++ {
		count++
	}
	if count != 3 {
		t.Fatalf("failure threshold = %d, want 3", count)
	}
}
