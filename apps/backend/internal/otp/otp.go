// Package otp implements the MFS ownership verification flow used during onboarding.
package otp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/config"
	dbpool "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/Seizmann/RexiO-Pay/backend/internal/phone"
	"github.com/Seizmann/RexiO-Pay/backend/internal/sms"
)

const (
	// Amount is the fixed amount merchants send to prove ownership.
	Amount int64 = 1

	statusPending  = "pending"
	statusVerified = "verified"
	statusExpired  = "expired"
)

var (
	ErrInvalidProvider    = errors.New("unsupported MFS provider")
	ErrInvalidAccountType = errors.New("unsupported MFS account type")
	ErrInvalidPhone       = errors.New("invalid MFS phone number")
	ErrInvalidAmount      = errors.New("OTP verification SMS amount must be exactly 1")
	ErrNoPendingOTP       = errors.New("no pending OTP verification")
	ErrPlatformDevice     = errors.New("SMS did not arrive from the platform verification device")
	ErrInvalidOTPID       = errors.New("invalid OTP verification ID")
)

// ParsedSMS is the small parser-to-OTP contract. The SMS parser may add fields
// without changing this package; only these fields are used by TryMatch.
// Amount is integer taka, as it is in the database and the rest of the backend.
type ParsedSMS struct {
	Provider         string `json:"provider"`
	AccountType      string `json:"account_type"`
	Amount           int64  `json:"amount"`
	SenderNumber     string `json:"sender_number"`
	TrxID            string `json:"trx_id"`
	SMSID            string `json:"sms_id,omitempty"`
	DeviceID         string `json:"device_id,omitempty"`
	MerchantID       string `json:"merchant_id,omitempty"`
	DeviceMerchantID string `json:"device_merchant_id,omitempty"`
}

// InitResult is returned after an OTP verification request is created.
type InitResult struct {
	dbsqlc.MfsOtpVerification
	VerificationNumber string `json:"verification_number"`
	Amount             int64  `json:"amount"`
}

// StatusResult is the merchant-scoped status response.
type StatusResult struct {
	dbsqlc.MfsOtpVerification
	VerificationNumber string `json:"verification_number"`
	Amount             int64  `json:"amount"`
}

// MatchResult contains both records changed by a successful match.
type MatchResult struct {
	Verification dbsqlc.MfsOtpVerification `json:"verification"`
	Profile      dbsqlc.PaymentProfile     `json:"profile"`
}

// Service coordinates OTP verification persistence and onboarding profile creation.
type Service struct {
	pool *dbpool.Pool
	cfg  *config.Config
	q    *dbsqlc.Queries
	now  func() time.Time
}

// New creates an OTP service backed by the application's sqlc database pool.
func New(pool *dbpool.Pool, cfg *config.Config) *Service {
	service := &Service{pool: pool, cfg: cfg, now: time.Now}
	if pool != nil && pool.SqlDB != nil {
		service.q = dbsqlc.New(pool.SqlDB)
	}
	return service
}

// NewService is an explicit alias for New.
func NewService(pool *dbpool.Pool, cfg *config.Config) *Service { return New(pool, cfg) }

// Init starts a verification using a one-shot service. Callers that handle
// multiple requests should retain a Service from New instead.
func Init(ctx context.Context, pool *dbpool.Pool, cfg *config.Config, merchantID, provider, accountType, mfsNumber string) (InitResult, error) {
	return New(pool, cfg).Init(ctx, merchantID, provider, accountType, mfsNumber)
}

// GetStatus reads a merchant-scoped verification using a one-shot service.
func GetStatus(ctx context.Context, pool *dbpool.Pool, cfg *config.Config, merchantID, id string) (StatusResult, error) {
	return New(pool, cfg).GetStatus(ctx, merchantID, id)
}

// TryMatch matches a parsed platform-device SMS using a one-shot service.
func TryMatch(ctx context.Context, pool *dbpool.Pool, cfg *config.Config, parsed ParsedSMS) (MatchResult, error) {
	return New(pool, cfg).TryMatch(ctx, parsed)
}

// Init starts a verification for a merchant's MFS number. The number is
// canonicalized before storage and the profile is not created until TryMatch.
func (s *Service) Init(ctx context.Context, merchantID, provider, accountType, mfsNumber string) (InitResult, error) {
	if s == nil || s.q == nil {
		return InitResult{}, errors.New("otp: database is not configured")
	}
	if strings.TrimSpace(merchantID) == "" {
		return InitResult{}, errors.New("merchant ID is required")
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	accountType = strings.ToLower(strings.TrimSpace(accountType))
	if !validProvider(provider) {
		return InitResult{}, ErrInvalidProvider
	}
	if !validAccountType(accountType) {
		return InitResult{}, ErrInvalidAccountType
	}
	mfsNumber = phone.Canonicalize(mfsNumber)
	if mfsNumber == "" {
		return InitResult{}, ErrInvalidPhone
	}

	verification, err := s.q.CreateOTPVerification(ctx, dbsqlc.CreateOTPVerificationParams{
		ID:          idgen.New(idgen.PrefixOTP),
		MerchantID:  merchantID,
		MfsNumber:   mfsNumber,
		Provider:    provider,
		AccountType: accountType,
	})
	if err != nil {
		return InitResult{}, fmt.Errorf("otp init: create verification: %w", err)
	}
	s.audit(ctx, verification.MerchantID, "otp.initiated", "mfs_otp_verification", verification.ID, map[string]any{
		"provider": provider, "account_type": accountType, "mfs_number": mfsNumber, "amount": Amount,
	})
	return InitResult{
		MfsOtpVerification: verification,
		VerificationNumber: s.verificationNumber(),
		Amount:             Amount,
	}, nil
}

// GetStatus returns a verification only when it belongs to merchantID.
func (s *Service) GetStatus(ctx context.Context, merchantID, id string) (StatusResult, error) {
	if s == nil || s.q == nil {
		return StatusResult{}, errors.New("otp: database is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return StatusResult{}, ErrInvalidOTPID
	}
	verification, err := s.q.GetOTPVerification(ctx, dbsqlc.GetOTPVerificationParams{ID: id, MerchantID: merchantID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StatusResult{}, ErrNoPendingOTP
		}
		return StatusResult{}, fmt.Errorf("otp status: get verification: %w", err)
	}
	// The generated status query intentionally does not mutate data. Reflect an
	// elapsed pending verification as expired until the next cleanup job runs.
	if verification.Status == statusPending && !verification.ExpiresAt.After(s.currentTime()) {
		verification.Status = statusExpired
	}
	return StatusResult{
		MfsOtpVerification: verification,
		VerificationNumber: s.verificationNumber(),
		Amount:             Amount,
	}, nil
}

// TryMatch consumes a platform-device SMS for the newest matching verification.
// A successful match creates the first payment profile with account_type copied
// from the verification and marks that profile as OTP verified.
func (s *Service) TryMatch(ctx context.Context, parsed ParsedSMS) (MatchResult, error) {
	if s == nil || s.q == nil {
		return MatchResult{}, errors.New("otp: database is not configured")
	}
	provider := strings.ToLower(strings.TrimSpace(parsed.Provider))
	if !validProvider(provider) {
		return MatchResult{}, ErrInvalidProvider
	}
	if parsed.Amount != Amount {
		return MatchResult{}, ErrInvalidAmount
	}
	if !s.isPlatformSMS(parsed) {
		return MatchResult{}, ErrPlatformDevice
	}
	sender := phone.Canonicalize(parsed.SenderNumber)
	if sender == "" {
		return MatchResult{}, ErrInvalidPhone
	}

	verification, err := s.q.GetPendingOTPByNumber(ctx, dbsqlc.GetPendingOTPByNumberParams{
		MfsNumber: sender,
		Provider:  provider,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MatchResult{}, ErrNoPendingOTP
		}
		return MatchResult{}, fmt.Errorf("otp match: get pending verification: %w", err)
	}
	if verification.Status != statusPending || !verification.ExpiresAt.After(s.currentTime()) {
		return MatchResult{}, ErrNoPendingOTP
	}
	if parsed.AccountType != "" && strings.ToLower(strings.TrimSpace(parsed.AccountType)) != verification.AccountType {
		return MatchResult{}, ErrInvalidAccountType
	}

	profile, err := s.q.CreateProfile(ctx, dbsqlc.CreateProfileParams{
		ID:                  idgen.New(idgen.PrefixProfile),
		MerchantID:          verification.MerchantID,
		Provider:            verification.Provider,
		AccountType:         verification.AccountType,
		MfsNumber:           verification.MfsNumber,
		DisplayName:         "",
		BalanceVerification: true,
		IsOtpVerified:       true,
	})
	if err != nil {
		return MatchResult{}, fmt.Errorf("otp match: create profile: %w", err)
	}
	if err := s.q.MarkOTPVerified(ctx, verification.ID); err != nil {
		return MatchResult{}, fmt.Errorf("otp match: mark verified: %w", err)
	}
	s.audit(ctx, verification.MerchantID, "otp.verified", "mfs_otp_verification", verification.ID, map[string]any{
		"profile_id": profile.ID, "sms_id": parsed.SMSID, "device_id": parsed.DeviceID,
		"provider": verification.Provider, "account_type": verification.AccountType, "amount": Amount,
	})
	verification.Status = statusVerified
	return MatchResult{Verification: verification, Profile: profile}, nil
}

func (s *Service) verificationNumber() string {
	if s != nil && s.cfg != nil {
		return phone.Canonicalize(s.cfg.PlatformVerificationNumber)
	}
	return ""
}

func (s *Service) isPlatformSMS(parsed ParsedSMS) bool {
	if s == nil || s.cfg == nil || strings.TrimSpace(s.cfg.PlatformDeviceMerchantID) == "" {
		return true
	}
	return parsed.MerchantID == s.cfg.PlatformDeviceMerchantID || parsed.DeviceMerchantID == s.cfg.PlatformDeviceMerchantID
}

func (s *Service) currentTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func validProvider(provider string) bool { return provider == "bkash" || provider == "nagad" }

func validAccountType(accountType string) bool {
	return accountType == sms.AccountPersonal || accountType == sms.AccountAgent || accountType == sms.AccountMerchant
}

// audit is deliberately best effort. The OTP/profile writes are already
// committed by separate sqlc calls, so reporting an audit insert failure as a
// failed match would cause clients to retry and create duplicate profiles.
func (s *Service) audit(ctx context.Context, merchantID, action, entity, entityID string, details map[string]any) {
	if s == nil || s.q == nil {
		return
	}
	payload, err := json.Marshal(details)
	if err != nil {
		slog.Error("otp: marshal audit details", "err", err)
		return
	}
	if err := s.q.InsertAuditLog(ctx, dbsqlc.InsertAuditLogParams{
		ID:         idgen.New(idgen.PrefixAudit),
		MerchantID: sql.NullString{String: merchantID, Valid: merchantID != ""},
		Action:     action,
		Entity:     entity,
		EntityID:   entityID,
		Details:    payload,
	}); err != nil {
		slog.Error("otp: write audit log", "action", action, "entity_id", entityID, "err", err)
	}
}
