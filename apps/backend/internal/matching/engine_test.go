package matching

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/Seizmann/RexiO-Pay/backend/internal/testutil"
)

func TestMatchingEngineIntegration(t *testing.T) {
	testutil.TestDB(t).Close()
	dsn := os.Getenv("TEST_DATABASE_URL")
	pool, err := dbpkg.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := requireMatchingSchema(pool); err != nil {
		t.Skipf("matching integration schema is not installed: %v", err)
	}

	tests := []struct {
		name string
		fn   func(*testing.T, *dbpkg.Pool)
	}{
		{"dedupe", testDedupe},
		{"balance mismatch", testBalanceMismatch},
		{"balance match", testBalanceMatch},
		{"trxid claim", testTrxIDClaim},
		{"unique unclaimed", testUniqueUnclaimed},
		{"ambiguous held", testAmbiguousHeld},
		{"late review", testLateReview},
		{"late accept", testLateAccept},
		{"no match", testNoMatch},
		{"reclaim held", testReclaimHeld},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.fn(t, pool) })
	}
}

func requireMatchingSchema(pool *dbpkg.Pool) error {
	var exists bool
	return pool.Raw.QueryRow(context.Background(), "SELECT to_regclass('public.checkout_sessions') IS NOT NULL").Scan(&exists)
}

type fixture struct {
	pool       *dbpkg.Pool
	merchantID string
	profileID  string
	engine     *Engine
	base       time.Time
}

func newFixture(t *testing.T, pool *dbpkg.Pool, settings map[string]any) *fixture {
	t.Helper()
	ctx := context.Background()
	merchantID, userID := idgen.New(idgen.PrefixMerchant), "usr_"+idgen.New("")
	profileID := idgen.New(idgen.PrefixProfile)
	settingsJSON, _ := json.Marshal(settings)
	_, err := pool.Raw.Exec(ctx, `INSERT INTO public.users (id,email) VALUES ($1,$2)`, userID, userID+"@example.test")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	_, err = pool.Raw.Exec(ctx, `INSERT INTO public.merchants (id,owner_user_id,name,settings) VALUES ($1,$2,$3,$4)`, merchantID, userID, "matching test", settingsJSON)
	if err != nil {
		t.Fatalf("insert merchant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Raw.Exec(context.Background(), `DELETE FROM public.merchants WHERE id=$1`, merchantID)
	})
	q := dbsqlc.New(pool.SqlDB)
	if _, err := q.CreateProfile(ctx, dbsqlc.CreateProfileParams{ID: profileID, MerchantID: merchantID, Provider: "bkash", AccountType: "personal", MfsNumber: "01700000000", DisplayName: "test", BalanceVerification: true, IsOtpVerified: true}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	base := time.Now().UTC().Truncate(time.Microsecond)
	return &fixture{pool: pool, merchantID: merchantID, profileID: profileID, engine: New(pool, nil), base: base}
}

func (f *fixture) session(t *testing.T, id string, amount int64, created, expires time.Time, sender, trx string) dbsqlc.CheckoutSession {
	t.Helper()
	q := dbsqlc.New(f.pool.SqlDB)
	s, err := q.CreateSession(context.Background(), dbsqlc.CreateSessionParams{ID: id, MerchantID: f.merchantID, PaymentProfileID: f.profileID, Source: "api", Amount: amount, Currency: "BDT", Customer: json.RawMessage(`{}`), Metadata: json.RawMessage(`{}`), ReturnUrl: "", CancelUrl: "", ExpiresAt: expires})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	_, err = f.pool.Raw.Exec(context.Background(), `UPDATE public.checkout_sessions SET created_at=$2, sender_number_claim=$3, trx_id_claim=$4, claim_submitted_at=$5 WHERE id=$1`, id, created, nullString(sender), nullString(trx), nullTime(created))
	if err != nil {
		t.Fatalf("prepare session: %v", err)
	}
	return s
}

func (f *fixture) sms(t *testing.T, parsed ParsedSMS) Result {
	t.Helper()
	result, err := f.engine.Ingest(context.Background(), IngestInput{MerchantID: f.merchantID, PaymentProfileID: f.profileID, RawText: "test sms", Source: "app", ReceivedAt: parsed.SmsTime, Parsed: parsed})
	if err != nil {
		t.Fatalf("ingest SMS: %v", err)
	}
	return result
}

func parsed(at time.Time, amount int64, trx string) ParsedSMS {
	return ParsedSMS{Provider: "bkash", AccountType: "personal", Amount: amount, SenderNumber: "+880 1700-000-001", TrxID: trx, SmsTime: at}
}

func testDedupe(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false})
	f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "")
	first := f.sms(t, parsed(f.base, 500, "DUP-1"))
	second := f.sms(t, parsed(f.base.Add(time.Second), 500, "DUP-1"))
	if first.MatchStatus != MatchMatched || second.MatchStatus != MatchDuplicate {
		t.Fatalf("statuses: first=%s second=%s", first.MatchStatus, second.MatchStatus)
	}
}

func testBalanceMismatch(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": true})
	q := dbsqlc.New(pool.SqlDB)
	if err := q.UpdateTrackedBalance(context.Background(), dbsqlc.UpdateTrackedBalanceParams{ID: f.profileID, TrackedBalance: sql.NullInt64{Int64: 1000, Valid: true}}); err != nil {
		t.Fatal(err)
	}
	f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "")
	balance := int64(1700)
	p := parsed(f.base, 500, "BAL-1")
	p.Balance = &balance
	if got := f.sms(t, p); got.MatchStatus != MatchSuspect || got.ReviewReason != "balance_mismatch" {
		t.Fatalf("got %+v", got)
	}
}

func testBalanceMatch(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": true})
	q := dbsqlc.New(pool.SqlDB)
	_ = q.UpdateTrackedBalance(context.Background(), dbsqlc.UpdateTrackedBalanceParams{ID: f.profileID, TrackedBalance: sql.NullInt64{Int64: 1000, Valid: true}})
	f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "")
	balance := int64(1500)
	p := parsed(f.base, 500, "BAL-2")
	p.Balance = &balance
	if got := f.sms(t, p); got.MatchStatus != MatchMatched {
		t.Fatalf("got %+v", got)
	}
}

func testTrxIDClaim(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false, "claim_window_minutes": 10})
	s := f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "CLAIM-1")
	if got := f.sms(t, parsed(f.base, 500, "CLAIM-1")); got.SessionID != s.ID || got.Verification != VerificationTrxID {
		t.Fatalf("got %+v", got)
	}
}

func testUniqueUnclaimed(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false})
	s := f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "")
	if got := f.sms(t, parsed(f.base, 500, "AUTO-1")); got.SessionID != s.ID || got.Verification != VerificationAuto {
		t.Fatalf("got %+v", got)
	}
}

func testAmbiguousHeld(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false})
	f.session(t, idgen.New("cs_"), 500, f.base.Add(-2*time.Minute), f.base.Add(time.Hour), "01700000001", "")
	f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "")
	if got := f.sms(t, parsed(f.base, 500, "HOLD-1")); got.MatchStatus != MatchHeld || got.ReviewReason != "needs_trxid" {
		t.Fatalf("got %+v", got)
	}
}

func testLateReview(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false, "auto_accept_late_match": false})
	s := f.session(t, idgen.New("cs_"), 500, f.base.Add(-2*time.Hour), f.base.Add(-time.Hour), "01700000001", "")
	got := f.sms(t, parsed(f.base, 500, "LATE-1"))
	if got.ReviewReason != "late_match" || got.SessionID != s.ID {
		t.Fatalf("got %+v", got)
	}
}

func testLateAccept(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false, "auto_accept_late_match": true})
	s := f.session(t, idgen.New("cs_"), 500, f.base.Add(-2*time.Hour), f.base.Add(-time.Hour), "01700000001", "")
	got := f.sms(t, parsed(f.base, 500, "LATE-2"))
	if got.SessionID != s.ID || got.Verification != VerificationLate {
		t.Fatalf("got %+v", got)
	}
}

func testNoMatch(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false})
	if got := f.sms(t, parsed(f.base, 500, "NONE-1")); got.MatchStatus != MatchUnmatched {
		t.Fatalf("got %+v", got)
	}
}

func testReclaimHeld(t *testing.T, pool *dbpkg.Pool) {
	f := newFixture(t, pool, map[string]any{"balance_verification": false})
	s := f.session(t, idgen.New("cs_"), 500, f.base.Add(-2*time.Minute), f.base.Add(time.Hour), "01700000001", "")
	f.session(t, idgen.New("cs_"), 500, f.base.Add(-time.Minute), f.base.Add(time.Hour), "01700000001", "")
	p := parsed(f.base, 500, "RECLAIM-1")
	if got := f.sms(t, p); got.MatchStatus != MatchHeld {
		t.Fatalf("initial %+v", got)
	}
	if err := f.engine.ClaimSession(context.Background(), s.ID, "01700000001", "RECLAIM-1"); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.Raw.QueryRow(context.Background(), `SELECT status FROM public.checkout_sessions WHERE id=$1`, s.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "succeeded" {
		t.Fatalf("session status=%s", status)
	}
}

func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func nullTime(v time.Time) any {
	if v.IsZero() {
		return nil
	}
	return v
}
