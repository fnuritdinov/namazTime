package ratelimit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// memCounter — счётчик в памяти вместо Redis.
type memCounter struct {
	counts map[string]int64
	err    error
}

func (m *memCounter) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.counts[key]++
	return m.counts[key], nil
}

func setup(counter Counter, now *time.Time) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	return Middleware(counter, Options{
		Limit: 3, Window: time.Minute,
		Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return *now },
	})(ok)
}

func get(h http.Handler, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/v1/cities", nil)
	req.RemoteAddr = ip + ":12345"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLimit(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 10, 0, time.UTC) // 10-я секунда минуты
	h := setup(&memCounter{counts: map[string]int64{}}, &now)

	for i := 1; i <= 3; i++ {
		if rec := get(h, "1.1.1.1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d: status %d", i, rec.Code)
		}
	}
	rec := get(h, "1.1.1.1")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: status %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "51" { // до 12:01:00 осталось 50 с (+1)
		t.Errorf("Retry-After = %s, want 51", got)
	}

	// Другой IP — свой счётчик
	if rec := get(h, "2.2.2.2"); rec.Code != http.StatusOK {
		t.Errorf("other IP: status %d", rec.Code)
	}

	// Новая минута — счётчик с нуля
	now = now.Add(time.Minute)
	if rec := get(h, "1.1.1.1"); rec.Code != http.StatusOK {
		t.Errorf("next window: status %d", rec.Code)
	}
}

func TestRedisDownAllows(t *testing.T) {
	now := time.Now()
	h := setup(&memCounter{err: errors.New("redis down")}, &now)
	if rec := get(h, "1.1.1.1"); rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200 when counter fails", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:4000"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.7")

	if got := clientIP(r, false); got != "10.0.0.5" {
		t.Errorf("without proxy: %s", got)
	}
	if got := clientIP(r, true); got != "203.0.113.7" {
		t.Errorf("with proxy: %s, want last address", got)
	}
}
