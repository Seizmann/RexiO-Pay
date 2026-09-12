package devices

import (
	"github.com/Seizmann/RexiO-Pay/backend/internal/config"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
	"github.com/go-chi/chi/v5"
)

// Routes wires the device API. Pairing is intentionally unauthenticated but
// protected by a one-time, expiring token; all other routes require HMAC.
func Routes(pool *dbpkg.Pool, cfg *config.Config, tg *telegram.Alerter, parser Parser) chi.Router {
	r := chi.NewRouter()
	if cfg == nil {
		return r
	}
	pair := NewPairHandler(pool, cfg.DeviceSecretKey)
	auth := NewAuthMiddleware(pool, cfg.DeviceSecretKey, tg)
	configHandler := &ConfigHandler{Config: ConfigResponse{
		LatestAppVersion: cfg.DeviceLatestAppVersion,
		APKURL:           cfg.DeviceAPKURL,
		MandatoryUpdate:  cfg.DeviceMandatoryUpdate,
		ParserVersion:    cfg.DeviceParserVersion,
		SenderIDs:        cfg.DeviceSenderIDs,
	}}
	r.Post("/pair", pair.ServeHTTP)
	protected := chi.NewRouter()
	protected.Use(middleware.ReadBody)
	protected.Use(auth.Middleware)
	protected.Use(middleware.DeviceRateLimit)
	protected.Post("/sms", NewSMSHandler(pool, parser).ServeHTTP)
	protected.Post("/heartbeat", (&HeartbeatHandler{Pool: pool}).ServeHTTP)
	protected.Get("/config", configHandler.ServeHTTP)
	r.Mount("/", protected)
	return r
}
