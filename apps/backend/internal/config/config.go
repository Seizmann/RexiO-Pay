package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
// The server fails fast at startup if any required field is missing.
type Config struct {
	AppEnv string
	Port   string
	AppURL string

	DatabaseURL string

	// Device secret encryption: AES-256-GCM key (32 bytes, loaded from 64-char hex env var).
	// Rotate = all devices must re-pair. Never store in DB or logs.
	DeviceSecretKey []byte

	// Telegram alerts
	TelegramBotToken    string
	TelegramAdminChatID string

	// R2 storage
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	// Webhook signing fallback (per-endpoint secrets override this)
	WebhookSigningDefaultSecret string

	// Session / matching windows
	SessionTTLMinutes  int
	ClaimWindowMinutes int

	// MFS OTP onboarding
	PlatformVerificationNumber string
	PlatformDeviceMerchantID   string

	// Android device API configuration
	DeviceLatestAppVersion string
	DeviceAPKURL           string
	DeviceMandatoryUpdate  bool
	DeviceParserVersion    string
	DeviceSenderIDs        map[string][]string
}

// Load reads and validates all environment variables. Panics on missing required fields.
func Load() *Config {
	cfg := &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("PORT", "8080"),
		AppURL: getEnv("APP_URL", "https://pay.rexio.pro"),

		DatabaseURL: requireEnv("DATABASE_URL"),

		TelegramBotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramAdminChatID: getEnv("TELEGRAM_ADMIN_CHAT_ID", ""),

		R2AccountID:       requireEnv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     requireEnv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: requireEnv("R2_SECRET_ACCESS_KEY"),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "rexio-pay-assets"),
		R2PublicURL:       requireEnv("R2_PUBLIC_URL"),

		WebhookSigningDefaultSecret: getEnv("WEBHOOK_SIGNING_DEFAULT_SECRET", ""),

		SessionTTLMinutes:  getEnvInt("SESSION_TTL_MINUTES", 30),
		ClaimWindowMinutes: getEnvInt("CLAIM_WINDOW_MINUTES", 10),

		PlatformVerificationNumber: getEnv("PLATFORM_VERIFICATION_NUMBER", ""),
		PlatformDeviceMerchantID:   getEnv("PLATFORM_DEVICE_MERCHANT_ID", ""),

		DeviceLatestAppVersion: getEnv("DEVICE_LATEST_APP_VERSION", ""),
		DeviceAPKURL:           getEnv("DEVICE_APK_URL", ""),
		DeviceMandatoryUpdate:  getEnvBool("DEVICE_MANDATORY_UPDATE", false),
		DeviceParserVersion:    getEnv("DEVICE_PARSER_VERSION", ""),
		DeviceSenderIDs:        parseSenderIDs(getEnv("DEVICE_SENDER_IDS", "")),
	}

	// Decode device secret encryption key (64 hex chars = 32 bytes)
	keyHex := requireEnv("DEVICE_SECRET_ENCRYPTION_KEY")
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil || len(keyBytes) != 32 {
		panic(fmt.Sprintf("DEVICE_SECRET_ENCRYPTION_KEY must be a 64-char hex string (32 bytes), got len=%d: %v", len(keyHex), err))
	}
	cfg.DeviceSecretKey = keyBytes

	return cfg
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

// parseSenderIDs accepts comma-separated provider=s1|s2 entries, for example
// "bkash=bKash,16247;nagad=NAGAD,16167". Unknown and empty entries are ignored.
func parseSenderIDs(value string) map[string][]string {
	result := make(map[string][]string)
	for _, providerEntry := range strings.Split(value, ";") {
		providerEntry = strings.TrimSpace(providerEntry)
		if providerEntry == "" {
			continue
		}
		parts := strings.SplitN(providerEntry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		provider := strings.ToLower(strings.TrimSpace(parts[0]))
		if provider == "" {
			continue
		}
		for _, sender := range strings.FieldsFunc(parts[1], func(r rune) bool { return r == ',' || r == '|' }) {
			sender = strings.TrimSpace(sender)
			if sender != "" {
				result[provider] = append(result[provider], sender)
			}
		}
	}
	return result
}
