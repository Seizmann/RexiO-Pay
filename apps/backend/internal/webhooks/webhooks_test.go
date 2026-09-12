package webhooks

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSubscribes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		events string
		want   bool
	}{
		{name: "matching event", events: `["payment.succeeded","payment.expired"]`, want: true},
		{name: "wildcard", events: `["*"]`, want: true},
		{name: "not subscribed", events: `["payment.expired"]`, want: false},
		{name: "invalid json", events: `{"event":"payment.succeeded"}`, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := subscribes(json.RawMessage(tt.events), "payment.succeeded"); got != tt.want {
				t.Fatalf("subscribes() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEnqueueOptionsValidation(t *testing.T) {
	if _, err := Enqueue(context.Background(), nil, EnqueueOptions{}); err != ErrNoPool {
		t.Fatalf("Enqueue(nil pool) error = %v, want %v", err, ErrNoPool)
	}
}
