// Package httpcache — HTTP-кеширование по ТЗ (§2.5): Cache-Control + ETag + 304 Not Modified.
package httpcache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Rule — сколько можно кешировать ответы для путей с этим префиксом.
type Rule struct {
	Prefix string
	MaxAge time.Duration
}

// Middleware добавляет к успешным GET-ответам Cache-Control и ETag
// и отвечает 304 Not Modified, если у клиента уже есть актуальная версия.
func Middleware(rules []Rule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rule, ok := match(rules, r.URL.Path)
			if !ok || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Даём handler'у записать ответ в буфер, а не сразу клиенту
			rec := &recorder{header: http.Header{}, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			for k, v := range rec.header {
				w.Header()[k] = v
			}

			// 2. Ошибки не кешируем — отдаём как есть
			if rec.status != http.StatusOK {
				w.WriteHeader(rec.status)
				_, _ = w.Write(rec.body.Bytes())
				return
			}

			// 3. ETag — «отпечаток» содержимого ответа
			etag := makeETag(rec.body.Bytes())
			w.Header().Set("ETag", etag)

			// Запросы с координатами — только кеш устройства (private), не CDN:
			// по ТЗ (§16) координаты нельзя сохранять, а CDN хранит URL вместе с query.
			visibility := "public"
			if q := r.URL.Query(); q.Has("lat") || q.Has("lon") {
				visibility = "private"
			}
			w.Header().Set("Cache-Control", fmt.Sprintf("%s, max-age=%d", visibility, int(rule.MaxAge.Seconds())))

			// 4. У клиента та же версия → 304 без тела
			if etagMatches(r.Header.Get("If-None-Match"), etag) {
				w.Header().Del("Content-Type")
				w.Header().Del("Content-Length")
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write(rec.body.Bytes())
			}
		})
	}
}

func match(rules []Rule, path string) (Rule, bool) {
	for _, rule := range rules {
		if strings.HasPrefix(path, rule.Prefix) {
			return rule, true
		}
	}
	return Rule{}, false
}

func makeETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}

// etagMatches разбирает If-None-Match: "a", W/"b", или "*".
func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

// recorder — ResponseWriter, который копит ответ в памяти.
type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header         { return r.header }
func (r *recorder) WriteHeader(status int)      { r.status = status }
func (r *recorder) Write(b []byte) (int, error) { return r.body.Write(b) }
