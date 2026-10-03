// Package ratelimit — ограничение числа запросов с одного IP (§16 ТЗ: 120 в минуту).
//
// Алгоритм «фиксированное окно»: на каждую минуту у IP свой счётчик в Redis.
// Ключ "rl:1.2.3.4:29345678" живёт чуть дольше минуты и удаляется сам.
package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Counter увеличивает счётчик и возвращает новое значение.
// В main — RedisCounter, в тестах — счётчик в памяти.
type Counter interface {
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// RedisCounter — счётчик в Redis: общий для всех копий сервера.
type RedisCounter struct{ RDB *redis.Client }

func (c RedisCounter) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	// INCR и EXPIRE в одной транзакции (MULTI/EXEC): ключ никогда не останется без срока жизни.
	// ExpireNX ставит срок только при первом запросе окна, дальше не продлевает.
	var incr *redis.IntCmd
	_, err := c.RDB.TxPipelined(ctx, func(p redis.Pipeliner) error {
		incr = p.Incr(ctx, key)
		p.ExpireNX(ctx, key, ttl)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

type Options struct {
	Limit      int                              // запросов в окно
	Window     time.Duration                    // длина окна, обычно минута
	TrustProxy bool                             // брать IP из X-Forwarded-For (только за своим прокси/CDN!)
	RequestID  func(ctx context.Context) string // для поля requestId в ошибке
	Log        *slog.Logger
	Now        func() time.Time // для тестов; nil — time.Now
}

// Middleware отвечает 429 rate_limited, если IP превысил лимит в текущем окне.
// Если Redis недоступен — пропускает запрос: лучше без лимита, чем без сервиса.
func Middleware(counter Counter, opt Options) func(http.Handler) http.Handler {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := opt.Now()
			window := now.Unix() / int64(opt.Window.Seconds())
			key := fmt.Sprintf("rl:%s:%d", clientIP(r, opt.TrustProxy), window)

			n, err := counter.Incr(r.Context(), key, opt.Window+10*time.Second)
			if err != nil {
				opt.Log.Warn("rate limit: counter unavailable, request allowed", "err", err)
				next.ServeHTTP(w, r)
				return
			}

			remaining := max(opt.Limit-int(n), 0)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(opt.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if int(n) > opt.Limit {
				// Сколько секунд до начала следующего окна
				windowEnd := time.Unix((window+1)*int64(opt.Window.Seconds()), 0)
				retry := int(windowEnd.Sub(now).Seconds()) + 1
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				writeError(w, r, opt)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP — IP клиента. X-Forwarded-For легко подделать, поэтому верим ему,
// только если сервер стоит за нашим прокси, и берём ПОСЛЕДНИЙ адрес — его дописал наш прокси.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeError(w http.ResponseWriter, r *http.Request, opt Options) {
	id := ""
	if opt.RequestID != nil {
		id = opt.RequestID(r.Context())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":      "rate_limited",
			"message":   "too many requests, retry later",
			"requestId": id,
		},
	})
}
