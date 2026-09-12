package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/config"
	dbpool "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	devpkg "github.com/Seizmann/RexiO-Pay/backend/internal/devices"
	"github.com/Seizmann/RexiO-Pay/backend/internal/sms"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx := context.Background()
	pool, err := dbpool.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database initialization failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		dbStatus := "ok"
		if err := pool.Raw.Ping(r.Context()); err != nil {
			status = http.StatusServiceUnavailable
			dbStatus = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"status":"ok","db":"` + dbStatus + `"}`))
	})

	tg := telegram.New(cfg.TelegramBotToken, cfg.TelegramAdminChatID)
	deviceParser := devpkg.SMSParserAdapter{Parser: sms.New()}
	r.Mount("/v1/device", devpkg.Routes(pool, cfg, tg, deviceParser))
	go (&devpkg.OfflineMonitor{Pool: pool, Telegram: tg}).Run(context.Background())

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("rexio-pay backend starting", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("backend stopped")
}
