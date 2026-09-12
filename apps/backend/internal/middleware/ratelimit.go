package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
)

type tokenBucket struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket() *tokenBucket {
	return &tokenBucket{buckets: make(map[string]*bucket)}
}

func (tb *tokenBucket) allow(key string, maxTokens, refillRate float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	b, ok := tb.buckets[key]
	if !ok {
		b = &bucket{tokens: maxTokens, maxTokens: maxTokens, refillRate: refillRate, lastRefill: time.Now()}
		tb.buckets[key] = b
	}
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

var (
	deviceRL = newTokenBucket()
	publicRL = newTokenBucket()
)

// DeviceRateLimit limits device SMS ingest to 30 req/min per device.
func DeviceRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := GetDeviceID(r.Context())
		if key == "" {
			key = r.RemoteAddr
		}
		if !deviceRL.allow(key, 30, 0.5) {
			apierr.Render(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PublicRateLimit limits public checkout endpoints to 60 req/min per IP.
func PublicRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = r.RemoteAddr
		}
		if !publicRL.allow(ip, 60, 1.0) {
			apierr.Render(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}
