package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	adminapi "github.com/Seizmann/RexiO-Pay/backend/internal/api/admin"
	merchantapi "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant"
	merchantapikeys "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/apikeys"
	merchantbranding "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/branding"
	merchantdevices "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/devices"
	merchantdomains "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/domains"
	merchantpaymentprofiles "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/paymentprofiles"
	merchantsettings "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/settings"
	merchantstorage "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/storage"
	merchantteammembers "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/teammembers"
	otpapi "github.com/Seizmann/RexiO-Pay/backend/internal/api/otp"
	"github.com/Seizmann/RexiO-Pay/backend/internal/config"
	dbpool "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	devpkg "github.com/Seizmann/RexiO-Pay/backend/internal/devices"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idempotency"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
	"github.com/Seizmann/RexiO-Pay/backend/internal/sms"
	"github.com/Seizmann/RexiO-Pay/backend/internal/storage"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
	"github.com/Seizmann/RexiO-Pay/backend/internal/webhooks"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const (
	workerRestartDelay  = 5 * time.Second
	otpExpiryInterval   = time.Minute
	idempotencyInterval = time.Hour
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("configuration initialization failed", "err", err)
		os.Exit(1)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	pool, err := dbpool.Open(startupCtx, cfg.DatabaseURL)
	startupCancel()
	if err != nil {
		slog.Error("database initialization failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	tg := telegram.New(cfg.TelegramBotToken, cfg.TelegramAdminChatID)
	merchantHandler := merchantapi.NewHandler(pool)
	merchantHandler.AppURL = cfg.AppURL
	if cfg.SessionTTLMinutes > 0 {
		merchantHandler.SessionTTL = time.Duration(cfg.SessionTTLMinutes) * time.Minute
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		status := http.StatusOK
		dbStatus := "ok"
		if err := pool.Raw.Ping(req.Context()); err != nil {
			status = http.StatusServiceUnavailable
			dbStatus = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"status":"ok","db":"%s"}`, dbStatus)
	})

	apiKeyAuth := middleware.APIKeyAuth(pool)
	storageCtx, storageCancel := context.WithTimeout(context.Background(), 10*time.Second)
	r2, storageErr := storage.New(storageCtx, storage.Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Bucket:          cfg.R2BucketName,
		PublicURL:       cfg.R2PublicURL,
	})
	storageCancel()
	if storageErr != nil {
		slog.Error("R2 storage initialization failed; storage routes disabled", "err", storageErr)
	}

	// Additional merchant resources use the same API-key boundary as the core
	// merchant handler. They are mounted before the catch-all core router.
	r.Route("/v1", func(v1 chi.Router) {
		v1.Use(apiKeyAuth)
		v1.Mount("/api-keys", merchantapikeys.NewHandler(pool).Routes())
		v1.Mount("/branding", merchantbranding.NewHandler(pool).Routes())
		v1.Mount("/devices", merchantdevices.NewHandler(pool).Routes())
		v1.Mount("/domain-whitelist", merchantdomains.NewHandler(pool).Routes())
		v1.Mount("/payment-profiles", merchantpaymentprofiles.NewHandler(pool).Routes())
		v1.Mount("/settings", merchantsettings.NewHandler(pool).Routes())
		v1.Mount("/team", merchantteammembers.NewHandler(pool).Routes())
		otpHandler := otpapi.New(pool, cfg)
		v1.Route("/onboarding/otp", otpHandler.Routes)
		if storageErr == nil {
			v1.Mount("/assets", merchantstorage.NewHandler(r2).Routes())
		}
	})

	// The merchant handler owns both the unauthenticated checkout endpoints and
	// the core /v1 sessions, payments, and payment-link endpoints.
	r.Mount("/", merchantHandler.Routes())

	deviceParser := devpkg.SMSParserAdapter{Parser: sms.NewParser()}
	r.Mount("/v1/device", devpkg.Routes(pool, cfg, tg, deviceParser))

	r.Route("/rexio-admin", func(adminRouter chi.Router) {
		adminRouter.Use(apiKeyAuth)
		adminRouter.Mount("/", adminapi.NewHandler(pool).Routes())
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	workerCtx, workerCancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	startWorkers(workerCtx, &workers, pool, tg)

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("rexio-pay backend starting", "port", cfg.Port, "env", cfg.AppEnv)
		serverErr <- server.ListenAndServe()
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	select {
	case sig := <-shutdownSignal:
		slog.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
		}
	}

	workerCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}

	workersDone := make(chan struct{})
	go func() {
		workers.Wait()
		close(workersDone)
	}()
	select {
	case <-workersDone:
	case <-shutdownCtx.Done():
		slog.Error("background workers did not stop before shutdown deadline")
	}
	slog.Info("backend stopped")
}

// loadConfig converts config.Load's fail-fast panic into a controlled startup
// error. Invalid or missing secrets must still prevent the server from starting.
func loadConfig() (cfg *config.Config, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("load config: %v", recovered)
			cfg = nil
		}
	}()
	return config.Load(), nil
}

func startWorkers(ctx context.Context, workers *sync.WaitGroup, pool *dbpool.Pool, tg *telegram.Alerter) {
	workers.Add(4)

	go func() {
		defer workers.Done()
		dispatcher := webhooks.NewDispatcher(pool)
		for {
			err := dispatcher.Run(ctx)
			if err == nil || errors.Is(err, context.Canceled) {
				return
			}
			slog.Error("webhook dispatcher stopped; retrying", "err", err)
			if !waitForWorkerRestart(ctx) {
				return
			}
		}
	}()

	go func() {
		defer workers.Done()
		(&devpkg.OfflineMonitor{Pool: pool, Telegram: tg}).Run(ctx)
	}()

	q := dbsqlc.New(pool.SqlDB)
	go runPeriodicWorker(ctx, workers, "OTP expiry", otpExpiryInterval, func(runCtx context.Context) error {
		return q.ExpireOldOTPs(runCtx)
	})
	go runPeriodicWorker(ctx, workers, "idempotency cleanup", idempotencyInterval, func(runCtx context.Context) error {
		return idempotency.Cleanup(runCtx, pool)
	})
}

func runPeriodicWorker(ctx context.Context, workers *sync.WaitGroup, name string, interval time.Duration, work func(context.Context) error) {
	defer workers.Done()

	run := func() {
		if err := work(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("background worker failed", "worker", name, "err", err)
		}
	}
	run()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func waitForWorkerRestart(ctx context.Context) bool {
	timer := time.NewTimer(workerRestartDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
