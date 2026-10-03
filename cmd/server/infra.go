package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"nTime/internal/storage"
)

// openPostgres подключается к базе, проверяет связь и применяет миграции.
func openPostgres(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	log.Info("connected to postgres")

	if err := storage.Migrate(url); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	log.Info("migrations applied")
	return db, nil
}

// openRedis подключается к Redis. Короткие таймауты: Redis должен отвечать быстро или не отвечать вовсе.
func openRedis(ctx context.Context, addr string, log *slog.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		MaxRetries:   1,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	log.Info("connected to redis")
	return rdb, nil
}
