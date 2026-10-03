// Package compress сжимает ответы gzip, если клиент это поддерживает (§2.1 ТЗ).
// JSON сжимается в 5–10 раз — быстрее на мобильном интернете.
package compress

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// Пул gzip-писателей: создавать новый на каждый запрос дорого (~250 КБ памяти).
var writers = sync.Pool{New: func() any {
	w, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
	return w
}}

func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding") // кеши (CDN) хранят сжатую и несжатую версии отдельно
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

// gzipWriter решает, сжимать ли, в момент WriteHeader — когда уже известен статус.
type gzipWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func (g *gzipWriter) WriteHeader(status int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true

	h := g.Header()
	// Сжатое тело — уже другие байты: сильный ETag становится слабым (W/), как делает nginx.
	// В 304 тоже, чтобы клиент видел тот же ETag, что получил в 200.
	if etag := h.Get("ETag"); etag != "" && !strings.HasPrefix(etag, "W/") {
		h.Set("ETag", "W/"+etag)
	}
	// 204 и 304 — без тела; уже сжатое не сжимаем повторно
	if status != http.StatusNoContent && status != http.StatusNotModified && h.Get("Content-Encoding") == "" {
		h.Set("Content-Encoding", "gzip")
		h.Del("Content-Length") // длина изменится после сжатия
		g.gz = writers.Get().(*gzip.Writer)
		g.gz.Reset(g.ResponseWriter)
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		// Go угадывает Content-Type по первым байтам тела — после сжатия угадает неверно,
		// поэтому определяем его сами по несжатым данным.
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.WriteHeader(http.StatusOK)
	}
	if g.gz == nil {
		return g.ResponseWriter.Write(b)
	}
	return g.gz.Write(b)
}

func (g *gzipWriter) close() {
	if g.gz != nil {
		_ = g.gz.Close() // дописывает конец gzip-потока
		writers.Put(g.gz)
	}
}

// Unwrap — чтобы http.ResponseController видел исходный ResponseWriter.
func (g *gzipWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }
