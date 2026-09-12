// Package billing enforces per-plan resource limits.
// Called at every write that consumes a plan-limited resource.
package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
)

// Resource keys matching keys in plans.limits jsonb.
const (
	ResourcePaymentProfiles  = "payment_profiles"
	ResourceDevices          = "devices"
	ResourceSessionsPerMonth = "sessions_per_month"
	ResourceWebhookEndpoints = "webhook_endpoints"
	ResourceTeamMembers      = "team_members"
)

// Limiter checks and enforces plan limits.
type Limiter struct {
	pool *dbpkg.Pool
}

// New creates a Limiter.
func New(pool *dbpkg.Pool) *Limiter {
	return &Limiter{pool: pool}
}

// Check returns an HTTP 402 apierr if the merchant has hit their plan limit
// for the given resource. A limit of 0 means unlimited.
func (l *Limiter) Check(ctx context.Context, w http.ResponseWriter, merchantID, resource string) bool {
	if err := l.check(ctx, merchantID, resource); err != nil {
		apierr.Render(w, http.StatusPaymentRequired, apierr.CodePlanLimitExceeded,
			fmt.Sprintf("plan limit reached for %s: upgrade your plan", resource))
		return false
	}
	return true
}

func (l *Limiter) check(ctx context.Context, merchantID, resource string) error {
	q := dbsqlc.New(l.pool.SqlDB)

	merchant, err := q.GetMerchant(ctx, merchantID)
	if err != nil {
		return fmt.Errorf("billing: get merchant: %w", err)
	}

	plan, err := q.GetPlan(ctx, merchant.PlanID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // unknown plan — allow (fail open)
		}
		return fmt.Errorf("billing: get plan: %w", err)
	}

	var limits map[string]int64
	if err := json.Unmarshal(plan.Limits, &limits); err != nil {
		return nil // malformed limits — fail open
	}

	limit, ok := limits[resource]
	if !ok || limit == 0 {
		return nil // unlimited or not configured
	}

	current, err := l.currentCount(ctx, merchantID, resource)
	if err != nil {
		return nil // fail open on count errors
	}

	if current >= limit {
		return fmt.Errorf("limit %d reached for %s (current: %d)", limit, resource, current)
	}
	return nil
}

func (l *Limiter) currentCount(ctx context.Context, merchantID, resource string) (int64, error) {
	var n int64
	var query string
	switch resource {
	case ResourcePaymentProfiles:
		query = "SELECT COUNT(*) FROM public.payment_profiles WHERE merchant_id = $1"
	case ResourceDevices:
		query = "SELECT COUNT(*) FROM public.devices WHERE merchant_id = $1 AND status != 'disabled'"
	case ResourceSessionsPerMonth:
		query = "SELECT session_count_current_period FROM public.merchants WHERE id = $1"
	case ResourceWebhookEndpoints:
		query = "SELECT COUNT(*) FROM public.webhook_endpoints WHERE merchant_id = $1 AND status = 'active'"
	case ResourceTeamMembers:
		query = "SELECT COUNT(*) FROM public.team_members WHERE merchant_id = $1"
	default:
		return 0, nil
	}
	row := l.pool.SqlDB.QueryRowContext(ctx, query, merchantID)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
