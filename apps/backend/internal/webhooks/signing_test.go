package webhooks

import (
	"errors"
	"testing"
	"time"
)

func TestSignMatchesStripeStyleVector(t *testing.T) {
	timestamp := time.Unix(1_700_000_000, 0)
	got := Sign("whsec_test", []byte(`{"id":"evt_123"}`), timestamp)
	want := "t=1700000000,v1=5a607119c8de704a1cff10c500951d1446f61db411d9ced87bfd2e4ae2d167ba"
	if got != want {
		t.Fatalf("Sign() = %q, want %q", got, want)
	}
}

func TestVerifySignature(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	payload := []byte(`{"hello":"world"}`)
	header := Sign("secret", payload, now)

	tests := []struct {
		name    string
		header  string
		wantErr error
	}{
		{name: "valid", header: header},
		{name: "valid with extra v1", header: header + ",v1=00"},
		{name: "bad signature", header: "t=1700000000,v1=00", wantErr: ErrInvalidSignature},
		{name: "expired", header: Sign("secret", payload, now.Add(-6*time.Minute)), wantErr: ErrExpiredSignature},
		{name: "malformed", header: "v1=00", wantErr: ErrMalformedHeader},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature("secret", payload, tt.header, now)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("VerifySignature() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("VerifySignature() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifySignatureAllowsClockTolerance(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, delta := range []time.Duration{-5 * time.Minute, 5 * time.Minute} {
		signedAt := now.Add(delta)
		if err := VerifySignature("secret", []byte("body"), Sign("secret", []byte("body"), signedAt), now); err != nil {
			t.Fatalf("delta %s: VerifySignature() error = %v", delta, err)
		}
	}
}
