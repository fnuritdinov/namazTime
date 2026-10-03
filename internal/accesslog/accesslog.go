// Package accesslog пишет одну строку лога на каждый запрос.
// По ТЗ (§16) координаты в логи не попадают: lat/lon заменяются на "***".
package accesslog

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Параметры, значения которых нельзя писать в лог
var secretParams = []string{"lat", "lon"}

func Middleware(log *slog.Logger, requestID func(context.Context) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			level := slog.LevelInfo
			if sw.status >= 500 {
				level = slog.LevelError
			}
			log.Log(r.Context(), level, "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", MaskQuery(r.URL.Query()),
				"status", sw.status,
				"bytes", sw.bytes,
				"durationMs", time.Since(start).Milliseconds(),
				"requestId", requestID(r.Context()),
				"appVersion", r.Header.Get("X-App-Version"), // §2.8 ТЗ: какие версии приложения живы
			)
		})
	}
}

// MaskQuery возвращает query-строку, где значения lat/lon заменены на ***.
func MaskQuery(q url.Values) string {
	for _, p := range secretParams {
		if q.Has(p) {
			q.Set(p, "***")
		}
	}
	s, _ := url.QueryUnescape(q.Encode()) // "***" и кириллица в логе читаемы
	return s
}

// statusWriter запоминает статус и размер ответа.
type statusWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (s *statusWriter) WriteHeader(status int) {
	if !s.wroteHeader {
		s.status, s.wroteHeader = status, true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	s.wroteHeader = true
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
