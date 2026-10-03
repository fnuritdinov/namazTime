package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // база часовых поясов внутри бинарника (+~450 КБ)

	"nTime/internal/config"
	"nTime/internal/device"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

// run — сценарий запуска сервера сверху вниз.
func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ctx отменится по Ctrl+C или docker stop — тогда сервер мягко остановится
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Инфраструктура
	db, err := openPostgres(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer db.Close()

	rdb, err := openRedis(ctx, cfg.RedisAddr, log)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// 2. Сервисы и API
	app := newApp(db, cfg)
	go device.RunCleanup(ctx, app.deviceRepo, 12, 24*time.Hour, log)

	// 3. Адреса и фильтры
	router := newRouter(app.api, db, rdb, log)
	handler := withMiddleware(router, cfg, rdb, app.devices, log)

	// 4. Сервер
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return serve(ctx, srv, log)
}
