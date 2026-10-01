package prayer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// CachedStore — Redis-кеш перед другим Store (обычно Postgres).
// Реализует тот же интерфейс Store, поэтому сервис не замечает разницы.
type CachedStore struct {
	next Store
	rdb  *redis.Client
	ttl  time.Duration
	log  *slog.Logger
}

const cacheOpTimeout = 100 * time.Millisecond

func NewCachedStore(next Store, rdb *redis.Client, ttl time.Duration, log *slog.Logger) *CachedStore {
	return &CachedStore{next: next, rdb: rdb, ttl: ttl, log: log}
}

// cacheKey: "prayer:v1:41.31:69.24:2026-10-01:3:1"
// v1 — версия формата: поменяем структуру → сменим на v2, старые ключи просто истекут.
func cacheKey(k Key) string {
	return fmt.Sprintf("prayer:v1:%.2f:%.2f:%s:%d:%d",
		k.Lat, k.Lon, k.Date.Format("2006-01-02"), k.Method, k.School)
}

func (c *CachedStore) Get(ctx context.Context, k Key) (DayTimings, error) {
	key := cacheKey(k)
	// 1. Пробуем Redis
	rctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	data, err := c.rdb.Get(rctx, key).Bytes()
	cancel()
	switch {
	case err == nil:
		var d DayTimings
		if err := json.Unmarshal(data, &d); err == nil {
			return d, nil // нашли в кеше
		}
		c.log.Warn("cache: bad data, ignoring", "key", key)
	case errors.Is(err, redis.Nil):
		// ключа нет — обычный промах кеша
	default:
		// Redis недоступен — не падаем, идём в Postgres
		c.log.Warn("cache get failed", "err", err)
	}

	// 2. Промах → идём дальше (в Postgres)
	d, err := c.next.Get(ctx, k)
	if err != nil {
		return DayTimings{}, err // в т.ч. ErrNotFound — сервис сам решит, идти ли в Aladhan
	}

	// 3. Нашли в Postgres → кладём в Redis на будущее
	c.set(ctx, key, d)
	return d, nil
}

func (c *CachedStore) Save(ctx context.Context, k Key, d DayTimings) error {
	if err := c.next.Save(ctx, k, d); err != nil {
		return err
	}
	c.set(ctx, cacheKey(k), d)
	return nil
}

// set кладёт значение в Redis. Ошибки только логируем: кеш — не критичная часть.
func (c *CachedStore) set(ctx context.Context, key string, d DayTimings) {
	data, err := json.Marshal(d)
	if err != nil {
		return
	}
	rctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()
	if err := c.rdb.Set(rctx, key, data, c.ttl).Err(); err != nil {
		c.log.Warn("cache set failed", "err", err)
	}
}
