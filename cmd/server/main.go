package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"nTime/internal/accesslog"
	"nTime/internal/compress"
	"nTime/internal/device"
	"nTime/internal/hijrimonth"
	"nTime/internal/httpcache"
	"nTime/internal/official"
	"nTime/internal/ratelimit"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // база часовых поясов внутри бинарника (+~450 КБ)

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"nTime/internal/city"
	"nTime/internal/config"
	"nTime/internal/handler"
	"nTime/internal/schedule"
	"nTime/internal/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Postgres
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgxpool.New: %w", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	log.Info("connected to postgres")

	if err := storage.Migrate(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("migrations applied")

	// Redis — кеш: должен отвечать быстро или не отвечать вовсе
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		MaxRetries:   1,
	})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	log.Info("connected to redis")

	// 1. Роутер
	mux := http.NewServeMux()

	// 2. Служебный health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{"status": "ok", "postgres": "ok", "redis": "ok"}
		code := http.StatusOK
		if err := db.Ping(r.Context()); err != nil {
			status["postgres"], status["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			status["redis"], status["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(status)
	})

	// 3. Зависимости
	cityRepo := city.NewRepository(db)
	officialRepo := official.NewRepository(db)
	hijriRepo := hijrimonth.NewRepository(db)
	scheduleSvc := schedule.NewService(cityRepo, officialRepo, hijriRepo)
	hijriSvc := hijrimonth.NewService(hijriRepo)
	deviceSvc := device.NewService(device.NewRepository(db))

	// 4. API из openapi.yaml — один раз, с префиксом /v1
	api := handler.NewServer(cityRepo, scheduleSvc, hijriSvc, deviceSvc, handler.AppInfo{
		MinSupportedVersion: cfg.MinAppVersion,
		LatestVersion:       cfg.LatestAppVersion,
		SupportURL:          cfg.SupportURL,
		FeatureSync:         cfg.FeatureSync,
		FeatureQuranSearch:  cfg.FeatureQuranSearch,
		ContentVersions:     cfg.ContentVersions,
	})
	strictHandler := handler.NewStrictHandlerWithOptions(api, nil, handler.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handler.RequestErrorHandler,
		ResponseErrorHandlerFunc: handler.InternalErrorHandler(log),
	})
	handler.HandlerWithOptions(strictHandler, handler.StdHTTPServerOptions{
		BaseURL:          "/v1",
		BaseRouter:       mux,
		ErrorHandlerFunc: handler.RequestErrorHandler,
	})

	// 5. HTTP-кеширование по ТЗ (§2.5): расписания — 1 час, справочники — сутки
	cache := httpcache.Middleware([]httpcache.Rule{
		{Prefix: "/v1/prayer-times", MaxAge: time.Hour},
		{Prefix: "/v1/config", MaxAge: time.Hour},
		{Prefix: "/v1/cities", MaxAge: 24 * time.Hour},
		{Prefix: "/v1/calculation-methods", MaxAge: 24 * time.Hour},
		{Prefix: "/v1/ramadan", MaxAge: time.Hour},
		{Prefix: "/v1/hijri", MaxAge: time.Hour},
	})
	limit := ratelimit.Middleware(ratelimit.RedisCounter{RDB: rdb}, ratelimit.Options{
		Limit:      cfg.RateLimitPerMinute,
		Window:     time.Minute,
		TrustProxy: cfg.TrustProxy,
		RequestID:  handler.RequestIDFrom,
		Log:        log,
	})
	logRequests := accesslog.Middleware(log, handler.RequestIDFrom)
	deviceAuth := device.Auth(deviceSvc, "/v1/devices/me", handler.RequestIDFrom, log)

	// 5. HTTP-сервер
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler.RequestID(logRequests(limit(http.MaxBytesHandler(compress.Gzip(cache(deviceAuth(mux))), 64<<10)))),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
