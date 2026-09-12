package devices

import (
	"context"
	"log/slog"
	"time"

	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
)

const OfflineAfter = 3 * time.Minute

// OfflineMonitor periodically transitions active devices whose last heartbeat
// is older than the three-minute SLA to offline and alerts the operator.
type OfflineMonitor struct {
	Pool     *dbpkg.Pool
	Telegram *telegram.Alerter
	Interval time.Duration
}

func (m *OfflineMonitor) Run(ctx context.Context) {
	if m == nil {
		return
	}
	interval := m.Interval
	if interval <= 0 {
		interval = time.Minute
	}
	m.check(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

func (m *OfflineMonitor) check(ctx context.Context) {
	if m == nil || m.Pool == nil {
		return
	}
	q := db.New(m.Pool.SqlDB)
	stale, err := q.ListStaleDevices(ctx)
	if err != nil {
		slog.Error("device offline monitor query failed", "err", err)
		return
	}
	for _, device := range stale {
		if err := q.UpdateDeviceStatus(ctx, db.UpdateDeviceStatusParams{ID: device.ID, Status: "offline"}); err != nil {
			slog.Error("device offline monitor update failed", "device_id", device.ID, "err", err)
			continue
		}
		if m.Telegram != nil {
			m.Telegram.Send("RexiO Pay: device offline — device_id: " + device.ID + " merchant_id: " + device.MerchantID)
		}
	}
}

// Check runs one monitor pass and is useful for scheduled jobs and tests.
func (m *OfflineMonitor) Check(ctx context.Context) { m.check(ctx) }
