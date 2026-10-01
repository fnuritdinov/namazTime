package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"nTime/internal/official"
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
	scheduleSvc := schedule.NewService(cityRepo, officialRepo)

	// 4. API из openapi.yaml — один раз, с префиксом /v1
	api := handler.NewServer(cityRepo, scheduleSvc)
	strictHandler := handler.NewStrictHandlerWithOptions(api, nil, handler.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handler.RequestErrorHandler,
		ResponseErrorHandlerFunc: handler.InternalErrorHandler(log),
	})
	handler.HandlerWithOptions(strictHandler, handler.StdHTTPServerOptions{
		BaseURL:          "/v1",
		BaseRouter:       mux,
		ErrorHandlerFunc: handler.RequestErrorHandler,
	})

	// 5. HTTP-сервер
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler.RequestID(mux),
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
