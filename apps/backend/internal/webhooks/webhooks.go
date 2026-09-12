package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
)

const (
	DefaultBatchSize      int32 = 25
	DefaultRequestTimeout       = 10 * time.Second
)

var ErrNoPool = errors.New("webhooks: nil database pool")

// Event is the envelope sent to an endpoint. Payload must be valid JSON.
type Event struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// EnqueueOptions describes an event to queue for all subscribed active
// endpoints belonging to a merchant.
type EnqueueOptions struct {
	MerchantID string
	EventType  string
	Payload    json.RawMessage
	EventID    string
}

// Enqueue creates one pending delivery for each active endpoint subscribed to
// event type. The event envelope is stored as JSON so retries resend the exact
// same bytes. It returns the number of queued deliveries.
func Enqueue(ctx context.Context, pool *dbpkg.Pool, opts EnqueueOptions) (int, error) {
	if pool == nil || pool.SqlDB == nil {
		return 0, ErrNoPool
	}
	if strings.TrimSpace(opts.MerchantID) == "" {
		return 0, errors.New("webhooks: merchant ID is required")
	}
	if strings.TrimSpace(opts.EventType) == "" {
		return 0, errors.New("webhooks: event type is required")
	}
	if !json.Valid(opts.Payload) {
		return 0, errors.New("webhooks: payload must be valid JSON")
	}
	if opts.EventID == "" {
		opts.EventID = idgen.New(idgen.PrefixWebhook)
	}

	q := dbsqlc.New(pool.SqlDB)
	endpoints, err := q.ListWebhookEndpoints(ctx, opts.MerchantID)
	if err != nil {
		return 0, fmt.Errorf("webhooks: list endpoints: %w", err)
	}
	queued := 0
	for _, endpoint := range endpoints {
		if endpoint.Status != "active" || !subscribes(endpoint.Events, opts.EventType) {
			continue
		}
		body, err := json.Marshal(Event{ID: opts.EventID, Type: opts.EventType, Data: opts.Payload})
		if err != nil {
			return queued, fmt.Errorf("webhooks: marshal event: %w", err)
		}
		if _, err := q.CreateWebhookDelivery(ctx, dbsqlc.CreateWebhookDeliveryParams{
			ID:         idgen.New(idgen.PrefixDelivery),
			EndpointID: endpoint.ID,
			MerchantID: opts.MerchantID,
			EventType:  opts.EventType,
			Payload:    body,
		}); err != nil {
			return queued, fmt.Errorf("webhooks: create delivery: %w", err)
		}
		queued++
	}
	return queued, nil
}

// EnqueueEvent is an alias for Enqueue.
func EnqueueEvent(ctx context.Context, pool *dbpkg.Pool, merchantID, eventType string, payload json.RawMessage) (int, error) {
	return Enqueue(ctx, pool, EnqueueOptions{MerchantID: merchantID, EventType: eventType, Payload: payload})
}

func subscribes(events json.RawMessage, eventType string) bool {
	var values []string
	if err := json.Unmarshal(events, &values); err != nil {
		return false
	}
	for _, value := range values {
		if value == eventType || value == "*" {
			return true
		}
	}
	return false
}

// Dispatcher delivers pending webhook rows until ctx is canceled. It is safe
// to run as a long-lived goroutine; each poll handles at most BatchSize rows.
type Dispatcher struct {
	Pool           *dbpkg.Pool
	Client         *http.Client
	BatchSize      int32
	PollInterval   time.Duration
	RequestTimeout time.Duration
	Now            func() time.Time
}

func NewDispatcher(pool *dbpkg.Pool) *Dispatcher {
	return &Dispatcher{
		Pool:           pool,
		Client:         &http.Client{Timeout: DefaultRequestTimeout},
		BatchSize:      DefaultBatchSize,
		PollInterval:   time.Minute,
		RequestTimeout: DefaultRequestTimeout,
		Now:            time.Now,
	}
}

// Run polls and dispatches pending deliveries until ctx is canceled. Database
// and network errors are returned; a failed HTTP delivery is persisted and
// does not stop the dispatcher.
func (d *Dispatcher) Run(ctx context.Context) error {
	if d == nil || d.Pool == nil || d.Pool.SqlDB == nil {
		return ErrNoPool
	}
	interval := d.PollInterval
	if interval <= 0 {
		interval = time.Minute
	}
	for {
		if err := d.DispatchOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// DispatchOnce delivers one batch of due deliveries.
func (d *Dispatcher) DispatchOnce(ctx context.Context) error {
	if d == nil || d.Pool == nil || d.Pool.SqlDB == nil {
		return ErrNoPool
	}
	limit := d.BatchSize
	if limit <= 0 {
		limit = DefaultBatchSize
	}
	q := dbsqlc.New(d.Pool.SqlDB)
	rows, err := q.GetPendingDeliveries(ctx, limit)
	if err != nil {
		return fmt.Errorf("webhooks: get pending deliveries: %w", err)
	}
	for _, row := range rows {
		if err := d.deliver(ctx, q, row); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) deliver(ctx context.Context, q *dbsqlc.Queries, row dbsqlc.GetPendingDeliveriesRow) error {
	timestamp := time.Now()
	if d.Now != nil {
		timestamp = d.Now()
	}
	signature := Sign(row.Secret, row.Payload, timestamp)
	reqCtx := ctx
	if d.RequestTimeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, d.RequestTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, row.Url, strings.NewReader(string(row.Payload)))
	if err != nil {
		return d.recordFailure(ctx, q, row, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(SignatureHeader, signature)
	req.Header.Set(EventHeader, row.EventType)
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return d.recordFailure(ctx, q, row, 0)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return q.UpdateDeliverySuccess(ctx, dbsqlc.UpdateDeliverySuccessParams{
			ID:               row.ID,
			LastResponseCode: sql.NullInt32{Int32: int32(resp.StatusCode), Valid: true},
		})
	}
	return d.recordFailure(ctx, q, row, int32(resp.StatusCode))
}

func (d *Dispatcher) recordFailure(ctx context.Context, q *dbsqlc.Queries, row dbsqlc.GetPendingDeliveriesRow, code int32) error {
	attempt := row.AttemptCount + 1
	now := time.Now()
	if d.Now != nil {
		now = d.Now()
	}
	next, status := NextRetry(now, attempt)
	responseCode := sql.NullInt32{}
	if code > 0 {
		responseCode = sql.NullInt32{Int32: code, Valid: true}
	}
	return q.UpdateDeliveryFailed(ctx, dbsqlc.UpdateDeliveryFailedParams{
		ID:               row.ID,
		NextRetryAt:      next,
		Status:           status,
		LastResponseCode: responseCode,
	})
}

// Redeliver resets a merchant-owned delivery to pending and clears its retry
// count, allowing the dispatcher to send it immediately.
func Redeliver(ctx context.Context, pool *dbpkg.Pool, deliveryID, merchantID string) error {
	if pool == nil || pool.SqlDB == nil {
		return ErrNoPool
	}
	return dbsqlc.New(pool.SqlDB).ResetDelivery(ctx, dbsqlc.ResetDeliveryParams{ID: deliveryID, MerchantID: merchantID})
}
