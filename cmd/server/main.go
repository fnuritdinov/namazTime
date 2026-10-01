package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"nTime/internal/city"
	"nTime/internal/config"
	"nTime/internal/geo"
	"nTime/internal/handler"
	"nTime/internal/prayer"
	"nTime/internal/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // база часовых поясов внутри бинарника (+~450 КБ)

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
		return fmt.Errorf("error from config.Load %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("error from pgxpool.New %w", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("error from db.Ping %w", err)
	}
	log.Info("connected to postgres")

	if err := storage.Migrate(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("migration applied")

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		MaxRetries:   1,
	})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error from %w", err)
	}
	log.Info("connected to redis")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{"status": "ok", "postgres": "ok", "redis": "ok"}
		code := http.StatusOK
		if err := db.Ping(r.Context()); err != nil {
			status["postgres"], status["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}

		if err = rdb.Ping(r.Context()).Err(); err != nil {
			status["redis"], status["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(status)
	})

	// Время намаза: Postgres + Aladhan
	prayerRepo := prayer.NewRepository(db)
	prayerStore := prayer.NewCachedStore(prayerRepo, rdb, 7*24*time.Hour, log)
	aladhan := prayer.NewAladhanClient(cfg.AladhanBaseURL)
	prayerSvc := prayer.NewService(prayerStore, aladhan, cfg.AladhanMethod, log)

	geoLoc, err := geo.New()
	if err != nil {
		return fmt.Errorf("geo: %w", err)
	}
	log.Info("geo data loaded")

	cityRepo := city.NewRepository(db)
	api := handler.NewServer(prayerSvc, geoLoc, cityRepo)
	strictHandler := handler.NewStrictHandlerWithOptions(api, nil, handler.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handler.RequestErrorHandler,
		ResponseErrorHandlerFunc: handler.InternalErrorHandler(log),
	})
	handler.HandlerWithOptions(strictHandler, handler.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: handler.RequestErrorHandler,
	})

	handler.HandlerWithOptions(strictHandler, handler.StdHTTPServerOptions{
		BaseURL:          "/v1",
		BaseRouter:       mux,
		ErrorHandlerFunc: handler.RequestErrorHandler,
	})

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
