// Package matching implements the SMS-to-checkout-session matching engine.
package matching

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sqlc-dev/pqtype"

	"github.com/Seizmann/RexiO-Pay/backend/internal/config"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/Seizmann/RexiO-Pay/backend/internal/phone"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
	"github.com/Seizmann/RexiO-Pay/backend/internal/webhooks"
)

const (
	MatchMatched   = "matched"
	MatchDuplicate = "duplicate"
	MatchSuspect   = "suspect"
	MatchHeld      = "held"
	MatchUnmatched = "unmatched"

	VerificationTrxID = "trxid_claim"
	VerificationAuto  = "auto"
	VerificationLate  = "late_auto"
)

var (
	ErrNoPool       = errors.New("matching: nil database pool")
	ErrInvalidInput = errors.New("matching: invalid SMS input")
)

// ParsedSMS is the parser/engine boundary. Parser implementations may fill
// SmsTime or ReceivedAt; SmsTime takes precedence when both are present.
type ParsedSMS struct {
	Provider     string    `json:"provider"`
	AccountType  string    `json:"account_type"`
	Amount       int64     `json:"amount"`
	SenderNumber string    `json:"sender_number"`
	TrxID        string    `json:"trx_id"`
	Balance      *int64    `json:"balance,omitempty"`
	SmsTime      time.Time `json:"sms_time"`
	ReceivedAt   time.Time `json:"received_at,omitempty"`
}

// IngestInput is used when the device endpoint has a raw SMS that has not yet
// been stored. The raw row is inserted before any parse/match work begins.
type IngestInput struct {
	ID               string
	MerchantID       string
	DeviceID         string
	PaymentProfileID string
	RawText          string
	Source           string
	SimSlot          int32
	ReceivedAt       time.Time
	Parsed           ParsedSMS
}

type Result struct {
	SMSID          string
	MatchStatus    string
	SessionID      string
	ReviewReason   string
	Verification   string
	DuplicateOfSMS string
}

type merchantSettings struct {
	ClaimWindowMinutes  int   `json:"claim_window_minutes"`
	AmountTolerance     int64 `json:"amount_tolerance"`
	AutoAcceptLateMatch bool  `json:"auto_accept_late_match"`
	BalanceVerification bool  `json:"balance_verification"`
}

// Engine coordinates matching and notifications. Config is optional; database
// merchant settings are authoritative, with Config providing safe defaults.
type Engine struct {
	Pool     *dbpkg.Pool
	Config   *config.Config
	Telegram *telegram.Alerter
	Now      func() time.Time
}

func New(pool *dbpkg.Pool, cfg *config.Config, alert ...*telegram.Alerter) *Engine {
	e := &Engine{Pool: pool, Config: cfg, Now: time.Now}
	if len(alert) > 0 {
		e.Telegram = alert[0]
	} else if cfg != nil {
		e.Telegram = telegram.New(cfg.TelegramBotToken, cfg.TelegramAdminChatID)
	}
	return e
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Ingest stores the raw SMS first, then parses/matches it. It is safe for an
// ingest handler to call this even when parsing has failed: callers can store a
// failed parse separately, while successful Parsed data follows this path.
func (e *Engine) Ingest(ctx context.Context, in IngestInput) (Result, error) {
	if e == nil || e.Pool == nil || e.Pool.SqlDB == nil {
		return Result{}, ErrNoPool
	}
	if in.MerchantID == "" || in.PaymentProfileID == "" || in.RawText == "" || in.Parsed.Provider == "" {
		return Result{}, ErrInvalidInput
	}
	if in.ID == "" {
		in.ID = idgen.New(idgen.PrefixSMS)
	}
	if in.Source == "" {
		in.Source = "app"
	}
	if in.ReceivedAt.IsZero() {
		in.ReceivedAt = e.now()
	}
	q := dbsqlc.New(e.Pool.SqlDB)
	_, err := q.InsertSMSMessage(ctx, dbsqlc.InsertSMSMessageParams{
		ID: in.ID, MerchantID: in.MerchantID,
		DeviceID: nullableString(in.DeviceID), PaymentProfileID: nullableString(in.PaymentProfileID),
		Provider: in.Parsed.Provider, AccountType: in.Parsed.AccountType,
		RawText: in.RawText, ParseStatus: "pending", MatchStatus: MatchUnmatched,
		Source: in.Source, SimSlot: nullableInt32(in.SimSlot), ReceivedAt: in.ReceivedAt,
	})
	if err != nil {
		return Result{}, fmt.Errorf("matching: insert SMS: %w", err)
	}
	result, err := e.Process(ctx, in.ID, in.Parsed)
	result.SMSID = in.ID
	return result, err
}

// Process parses an already-stored SMS row and runs the complete matching
// algorithm. It is useful to device handlers that persist raw SMS rows first.
func (e *Engine) Process(ctx context.Context, smsID string, parsed ParsedSMS) (Result, error) {
	if e == nil || e.Pool == nil || e.Pool.SqlDB == nil {
		return Result{}, ErrNoPool
	}
	if smsID == "" || parsed.Provider == "" || parsed.Amount < 0 || phone.Canonicalize(parsed.SenderNumber) == "" {
		return Result{}, ErrInvalidInput
	}
	if parsed.SmsTime.IsZero() {
		parsed.SmsTime = parsed.ReceivedAt
	}
	if parsed.SmsTime.IsZero() {
		parsed.SmsTime = e.now()
	}
	parsed.SenderNumber = phone.Canonicalize(parsed.SenderNumber)
	parsed.Provider = strings.ToLower(strings.TrimSpace(parsed.Provider))

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	tx, err := e.Pool.SqlDB.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, fmt.Errorf("matching: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbsqlc.New(tx)
	row, err := q.GetSMSMessage(ctx, smsID)
	if err != nil {
		return Result{}, fmt.Errorf("matching: get SMS: %w", err)
	}
	result := Result{SMSID: smsID}
	if row.MatchStatus == MatchMatched && row.MatchedSessionID.Valid {
		result.MatchStatus = MatchMatched
		result.SessionID = row.MatchedSessionID.String
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		return result, nil
	}

	// Check before marking this row parsed. A parsed row with the same ID is a
	// retry, not a duplicate of itself.
	if parsed.TrxID != "" {
		duplicate, dupErr := q.FindDuplicateTrxID(ctx, dbsqlc.FindDuplicateTrxIDParams{Provider: parsed.Provider, TrxID: parsed.TrxID})
		if dupErr == nil && duplicate != "" && duplicate != smsID {
			// Keep the duplicate row pending parse: the partial unique index permits
			// only one parsed row per transaction ID. The raw duplicate remains
			// retained and is classified for the review inbox.
			if err := q.UpdateSMSMatchStatus(ctx, dbsqlc.UpdateSMSMatchStatusParams{ID: smsID, MatchStatus: MatchDuplicate, MatchedSessionID: sql.NullString{}}); err != nil {
				return Result{}, fmt.Errorf("matching: mark duplicate: %w", err)
			}
			result.MatchStatus, result.DuplicateOfSMS = MatchDuplicate, duplicate
			if err := tx.Commit(); err != nil {
				return Result{}, err
			}
			return result, nil
		}
		if dupErr != nil && !errors.Is(dupErr, sql.ErrNoRows) {
			return Result{}, fmt.Errorf("matching: dedupe: %w", dupErr)
		}
	}
	if err := q.UpdateSMSParsed(ctx, dbsqlc.UpdateSMSParsedParams{ID: smsID, Parsed: parsedJSON(parsed), Provider: parsed.Provider, AccountType: parsed.AccountType}); err != nil {
		return Result{}, fmt.Errorf("matching: update parsed SMS: %w", err)
	}

	profileID := row.PaymentProfileID
	if !profileID.Valid || profileID.String == "" {
		return e.finishUnmatched(ctx, tx, q, smsID, result, "unmatched")
	}
	profile, err := q.GetProfile(ctx, dbsqlc.GetProfileParams{ID: profileID.String, MerchantID: row.MerchantID})
	if err != nil {
		return Result{}, fmt.Errorf("matching: get profile: %w", err)
	}
	merchant, err := q.GetMerchant(ctx, row.MerchantID)
	if err != nil {
		return Result{}, fmt.Errorf("matching: get merchant: %w", err)
	}
	settings := e.settings(merchant.Settings)
	if err := e.verifyBalance(ctx, q, row, profile, parsed, settings); err != nil {
		if errors.Is(err, errBalanceMismatch) {
			result.MatchStatus, result.ReviewReason = MatchSuspect, "balance_mismatch"
			if err := tx.Commit(); err != nil {
				return Result{}, err
			}
			return result, nil
		}
		return Result{}, err
	}

	lower, upper := parsed.Amount-settings.AmountTolerance, parsed.Amount+settings.AmountTolerance
	if lower < 0 {
		lower = 0
	}
	candidates, err := q.FindMatchCandidates(ctx, dbsqlc.FindMatchCandidatesParams{MerchantID: row.MerchantID, PaymentProfileID: profile.ID, Amount: lower, Amount_2: upper, SenderNumberClaim: sql.NullString{String: parsed.SenderNumber, Valid: true}})
	if err != nil {
		return Result{}, fmt.Errorf("matching: find candidates: %w", err)
	}
	// Include expired candidates for the 24-hour late-match branch. The query
	// already bounds them to the grace period relative to database now.
	expired, err := q.FindExpiredMatchCandidates(ctx, dbsqlc.FindExpiredMatchCandidatesParams{MerchantID: row.MerchantID, PaymentProfileID: profile.ID, Amount: lower, Amount_2: upper, SenderNumberClaim: sql.NullString{String: parsed.SenderNumber, Valid: true}})
	if err != nil {
		return Result{}, fmt.Errorf("matching: find expired candidates: %w", err)
	}
	candidates = appendCandidates(candidates, expired)

	// Priority 1: an explicit TrxID claim must match both trx and sender.
	for _, candidate := range candidates {
		if !candidate.TrxIDClaim.Valid || candidate.TrxIDClaim.String != parsed.TrxID || parsed.TrxID == "" || !candidate.ClaimSubmittedAt.Valid {
			continue
		}
		window := time.Duration(settings.ClaimWindowMinutes) * time.Minute
		if !inRange(parsed.SmsTime, candidate.ClaimSubmittedAt.Time.Add(-5*time.Minute), candidate.ClaimSubmittedAt.Time.Add(window)) {
			continue
		}
		if candidate.ExpiresAt.Before(parsed.SmsTime) {
			if !settings.AutoAcceptLateMatch {
				return e.holdLate(ctx, tx, q, smsID, candidate, result)
			}
			return e.verify(ctx, tx, q, row, candidate, profile, parsed, VerificationLate, result)
		}
		return e.verify(ctx, tx, q, row, candidate, profile, parsed, VerificationTrxID, result)
	}

	// Priority 2: an SMS in the session lifetime can be auto-verified only if
	// exactly one unclaimed session remains.
	unclaimed := make([]dbsqlc.CheckoutSession, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.TrxIDClaim.Valid {
			continue
		}
		if !candidate.ExpiresAt.Before(parsed.SmsTime) && !parsed.SmsTime.Before(candidate.CreatedAt) {
			unclaimed = append(unclaimed, candidate)
		}
	}
	if len(unclaimed) == 1 {
		return e.verify(ctx, tx, q, row, unclaimed[0], profile, parsed, VerificationAuto, result)
	}
	if len(unclaimed) > 1 {
		if err := q.UpdateSMSMatchStatus(ctx, dbsqlc.UpdateSMSMatchStatusParams{ID: smsID, MatchStatus: MatchHeld, MatchedSessionID: sql.NullString{}}); err != nil {
			return Result{}, err
		}
		for _, candidate := range unclaimed {
			if err := q.UpdateSessionNeedsReview(ctx, dbsqlc.UpdateSessionNeedsReviewParams{ID: candidate.ID, NeedsReview: true, ReviewReason: sql.NullString{String: "needs_trxid", Valid: true}}); err != nil {
				return Result{}, err
			}
		}
		result.MatchStatus, result.ReviewReason = MatchHeld, "needs_trxid"
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		return result, nil
	}

	// Priority 3: one expired candidate can be accepted automatically or sent
	// to review. A session outside the 24-hour query is intentionally ignored.
	late := make([]dbsqlc.CheckoutSession, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ExpiresAt.Before(parsed.SmsTime) && !parsed.SmsTime.After(candidate.ExpiresAt.Add(24*time.Hour)) {
			late = append(late, candidate)
		}
	}
	if len(late) == 1 {
		if settings.AutoAcceptLateMatch {
			return e.verify(ctx, tx, q, row, late[0], profile, parsed, VerificationLate, result)
		}
		return e.holdLate(ctx, tx, q, smsID, late[0], result)
	}
	return e.finishUnmatched(ctx, tx, q, smsID, result, MatchUnmatched)
}

// Match is an alias kept for callers whose ingest flow already stores and
// parses its SMS row.
func (e *Engine) Match(ctx context.Context, smsID string, parsed ParsedSMS) (Result, error) {
	return e.Process(ctx, smsID, parsed)
}

// ClaimSession stores a checkout claim and immediately retries held SMS rows
// for that profile. This implements the late "I have paid"/TrxID path.
func (e *Engine) ClaimSession(ctx context.Context, sessionID, senderNumber, trxID string) error {
	if e == nil || e.Pool == nil || e.Pool.SqlDB == nil {
		return ErrNoPool
	}
	sender := phone.Canonicalize(senderNumber)
	if sender == "" || strings.TrimSpace(trxID) == "" {
		return ErrInvalidInput
	}
	q := dbsqlc.New(e.Pool.SqlDB)
	session, err := q.GetSessionForPublicCheckout(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("matching: get claimed session: %w", err)
	}
	if err := q.UpdateSessionClaim(ctx, dbsqlc.UpdateSessionClaimParams{ID: sessionID, SenderNumberClaim: sql.NullString{String: sender, Valid: true}, TrxIDClaim: sql.NullString{String: strings.TrimSpace(trxID), Valid: true}}); err != nil {
		return fmt.Errorf("matching: save claim: %w", err)
	}
	if session.Amount < 0 {
		return ErrInvalidInput
	}
	held, err := q.GetHeldSMSForProfile(ctx, dbsqlc.GetHeldSMSForProfileParams{PaymentProfileID: sql.NullString{String: session.PaymentProfileID, Valid: true}, SenderNumber: sender, Amount: session.Amount})
	if err != nil {
		return fmt.Errorf("matching: find held SMS: %w", err)
	}
	for _, sms := range held {
		parsed, ok := parseStoredSMS(sms.Parsed)
		if !ok || parsed.TrxID != strings.TrimSpace(trxID) {
			continue
		}
		if _, err := e.Process(ctx, sms.ID, parsed); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) settings(raw json.RawMessage) merchantSettings {
	s := merchantSettings{ClaimWindowMinutes: 10, BalanceVerification: true}
	if e.Config != nil && e.Config.ClaimWindowMinutes > 0 {
		s.ClaimWindowMinutes = e.Config.ClaimWindowMinutes
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	if s.ClaimWindowMinutes <= 0 {
		s.ClaimWindowMinutes = 10
	}
	return s
}

var errBalanceMismatch = errors.New("matching: balance mismatch")

func (e *Engine) verifyBalance(ctx context.Context, q *dbsqlc.Queries, row dbsqlc.SmsMessage, profile dbsqlc.PaymentProfile, parsed ParsedSMS, settings merchantSettings) error {
	if !settings.BalanceVerification || !profile.BalanceVerification || parsed.Balance == nil {
		return nil
	}
	if !profile.TrackedBalance.Valid || profile.TrackedBalance.Int64+parsed.Amount != *parsed.Balance {
		_ = q.UpdateSMSMatchStatus(ctx, dbsqlc.UpdateSMSMatchStatusParams{ID: row.ID, MatchStatus: MatchSuspect, MatchedSessionID: sql.NullString{}})
		if e.Telegram != nil {
			e.Telegram.Send(fmt.Sprintf("RexiO Pay balance mismatch: merchant %s, profile %s, SMS %s", row.MerchantID, profile.ID, row.ID))
		}
		return errBalanceMismatch
	}
	if err := q.UpdateTrackedBalance(ctx, dbsqlc.UpdateTrackedBalanceParams{ID: profile.ID, TrackedBalance: sql.NullInt64{Int64: *parsed.Balance, Valid: true}}); err != nil {
		return fmt.Errorf("matching: update tracked balance: %w", err)
	}
	return nil
}

func (e *Engine) verify(ctx context.Context, tx *sql.Tx, q *dbsqlc.Queries, sms dbsqlc.SmsMessage, session dbsqlc.CheckoutSession, profile dbsqlc.PaymentProfile, parsed ParsedSMS, source string, result Result) (Result, error) {
	balance := sql.NullInt64{}
	if parsed.Balance != nil {
		balance = sql.NullInt64{Int64: *parsed.Balance, Valid: true}
	}
	payment, err := q.InsertPayment(ctx, dbsqlc.InsertPaymentParams{ID: idgen.New(idgen.PrefixPayment), MerchantID: session.MerchantID, SessionID: session.ID, SmsID: sql.NullString{String: sms.ID, Valid: true}, Provider: parsed.Provider, AccountType: parsed.AccountType, SenderNumber: parsed.SenderNumber, TrxID: parsed.TrxID, Amount: parsed.Amount, BalanceAfter: balance, VerifiedAt: e.now(), VerificationSource: source})
	if errors.Is(err, sql.ErrNoRows) {
		// A retry after a successful verification is idempotent. Continue with
		// the status updates so a previously interrupted request can converge.
	} else if err != nil {
		return Result{}, fmt.Errorf("matching: insert payment: %w", err)
	}
	if err := q.UpdateSessionStatus(ctx, dbsqlc.UpdateSessionStatusParams{ID: session.ID, Status: "succeeded"}); err != nil {
		return Result{}, err
	}
	if err := q.UpdateSessionNeedsReview(ctx, dbsqlc.UpdateSessionNeedsReviewParams{ID: session.ID, NeedsReview: false, ReviewReason: sql.NullString{}}); err != nil {
		return Result{}, err
	}
	if err := q.UpdateSMSMatchStatus(ctx, dbsqlc.UpdateSMSMatchStatusParams{ID: sms.ID, MatchStatus: MatchMatched, MatchedSessionID: sql.NullString{String: session.ID, Valid: true}}); err != nil {
		return Result{}, err
	}
	details, _ := json.Marshal(map[string]any{"sms_id": sms.ID, "trx_id": parsed.TrxID, "verification_source": source})
	if err := q.InsertAuditLog(ctx, dbsqlc.InsertAuditLogParams{ID: idgen.New(idgen.PrefixAudit), MerchantID: sql.NullString{String: session.MerchantID, Valid: true}, Action: "payment_verified", Entity: "payment", EntityID: payment.ID, Details: details}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Result{}, err
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("matching: commit verification: %w", err)
	}
	result.MatchStatus, result.SessionID, result.Verification = MatchMatched, session.ID, source
	payload, _ := json.Marshal(map[string]any{"session_id": session.ID, "payment_id": payment.ID, "amount": parsed.Amount, "provider": parsed.Provider, "account_type": parsed.AccountType, "sender_number": parsed.SenderNumber, "trx_id": parsed.TrxID, "verified_at": e.now(), "verification_source": source})
	if _, err := webhooks.Enqueue(context.Background(), e.Pool, webhooks.EnqueueOptions{MerchantID: session.MerchantID, EventType: "payment.succeeded", Payload: payload}); err != nil {
		slog.Warn("matching: enqueue webhook", "err", err, "session_id", session.ID)
	}
	return result, nil
}

func (e *Engine) holdLate(ctx context.Context, tx *sql.Tx, q *dbsqlc.Queries, smsID string, session dbsqlc.CheckoutSession, result Result) (Result, error) {
	if err := q.UpdateSMSMatchStatus(ctx, dbsqlc.UpdateSMSMatchStatusParams{ID: smsID, MatchStatus: MatchUnmatched, MatchedSessionID: sql.NullString{}}); err != nil {
		return Result{}, err
	}
	if err := q.UpdateSessionNeedsReview(ctx, dbsqlc.UpdateSessionNeedsReviewParams{ID: session.ID, NeedsReview: true, ReviewReason: sql.NullString{String: "late_match", Valid: true}}); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(); err != nil {
		return Result{}, err
	}
	result.MatchStatus, result.SessionID, result.ReviewReason = MatchUnmatched, session.ID, "late_match"
	return result, nil
}

func (e *Engine) finishUnmatched(ctx context.Context, tx *sql.Tx, q *dbsqlc.Queries, smsID string, result Result, status string) (Result, error) {
	if err := q.UpdateSMSMatchStatus(ctx, dbsqlc.UpdateSMSMatchStatusParams{ID: smsID, MatchStatus: status, MatchedSessionID: sql.NullString{}}); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(); err != nil {
		return Result{}, err
	}
	result.MatchStatus = status
	return result, nil
}

func appendCandidates(a, b []dbsqlc.CheckoutSession) []dbsqlc.CheckoutSession {
	seen := make(map[string]bool, len(a)+len(b))
	for _, c := range a {
		seen[c.ID] = true
	}
	for _, c := range b {
		if !seen[c.ID] {
			a = append(a, c)
			seen[c.ID] = true
		}
	}
	return a
}

func inRange(value, start, end time.Time) bool { return !value.Before(start) && !value.After(end) }
func nullableString(v string) sql.NullString   { return sql.NullString{String: v, Valid: v != ""} }
func nullableInt32(v int32) sql.NullInt32      { return sql.NullInt32{Int32: v, Valid: v != 0} }
func parsedJSON(p ParsedSMS) pqtype.NullRawMessage {
	b, _ := json.Marshal(p)
	return pqtype.NullRawMessage{RawMessage: b, Valid: true}
}

func parseStoredSMS(raw pqtype.NullRawMessage) (ParsedSMS, bool) {
	if !raw.Valid || len(raw.RawMessage) == 0 {
		return ParsedSMS{}, false
	}
	var p ParsedSMS
	if err := json.Unmarshal(raw.RawMessage, &p); err != nil || p.Provider == "" {
		return ParsedSMS{}, false
	}
	return p, true
}

// parserBalance supports parser implementations that represent an optional
// balance as either *int64, int64, or a nullable integer.
