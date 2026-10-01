package prayer

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// countingStore — заглушка «Postgres», которая считает обращения.
// Нужна, чтобы проверить: второй Get НЕ дошёл до Postgres, а ответил Redis.
type countingStore struct {
	fakeStore
	gets int
}

func (c *countingStore) Get(ctx context.Context, k Key) (DayTimings, error) {
	c.gets++
	return c.fakeStore.Get(ctx, k)
}

func newTestCache(t *testing.T) (*CachedStore, *countingStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t) // Redis в памяти, сам закроется после теста
	rdb := redis.NewClient(&redis.Options{
		Addr:        mr.Addr(),
		DialTimeout: 200 * time.Millisecond,
		MaxRetries:  1,
	})
	t.Cleanup(func() { rdb.Close() })

	inner := &countingStore{fakeStore: fakeStore{data: map[Key]DayTimings{}}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil)) // логи в тестах не нужны
	return NewCachedStore(inner, rdb, time.Hour, log), inner, mr
}

// Тест 1: после Save данные берутся из Redis, Postgres не трогается.
func TestCachedStore_GetFromRedis(t *testing.T) {
	cache, inner, mr := newTestCache(t)
	ctx := context.Background()

	k := Key{Lat: 41.31, Lon: 69.24, Date: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Method: 3, School: 1}
	want := DayTimings{Date: k.Date, Source: "aladhan", Timings: Timings{Fajr: "04:47"}}

	if err := cache.Save(ctx, k, want); err != nil {
		t.Fatal(err)
	}

	got, err := cache.Get(ctx, k)
	if err != nil {
		t.Fatal(err)
	}
	if got.Timings.Fajr != "04:47" {
		t.Errorf("Fajr = %q, want 04:47", got.Timings.Fajr)
	}
	if inner.gets != 0 {
		t.Errorf("Get дошёл до Postgres %d раз, ожидали 0 — должен был ответить Redis", inner.gets)
	}
	if ttl := mr.TTL(cacheKey(k)); ttl <= 0 {
		t.Errorf("TTL не выставлен: %v", ttl)
	}
}

// Тест 2: промах в Redis → берём из Postgres и кладём в Redis.
func TestCachedStore_MissThenFill(t *testing.T) {
	cache, inner, mr := newTestCache(t)
	ctx := context.Background()

	k := Key{Lat: 41.31, Lon: 69.24, Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), Method: 3, School: 1}
	inner.data[k] = DayTimings{Date: k.Date, Source: "aladhan"} // есть только в «Postgres»

	// Первый Get: в Redis нет → идём в Postgres
	if _, err := cache.Get(ctx, k); err != nil {
		t.Fatal(err)
	}
	// Второй Get: уже должен быть в Redis
	if _, err := cache.Get(ctx, k); err != nil {
		t.Fatal(err)
	}

	if inner.gets != 1 {
		t.Errorf("Postgres вызван %d раз, ожидали 1", inner.gets)
	}
	if !mr.Exists(cacheKey(k)) {
		t.Error("после промаха значение должно было попасть в Redis")
	}
}

// Тест 3: Redis упал → сервис всё равно отвечает из Postgres.
func TestCachedStore_RedisDown(t *testing.T) {
	cache, inner, mr := newTestCache(t)

	k := Key{Lat: 41.31, Lon: 69.24, Date: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), Method: 3, School: 1}
	inner.data[k] = DayTimings{Source: "aladhan"}

	mr.Close() // «роняем» Redis

	if _, err := cache.Get(context.Background(), k); err != nil {
		t.Fatalf("при упавшем Redis должны были ответить из Postgres, а получили ошибку: %v", err)
	}
}
