package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var body = strings.Repeat(`{"name":"Душанбе","timeZoneId":"Asia/Dushanbe"},`, 100)

func handler(status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = io.WriteString(w, body)
		}
	})
}

func TestGzip(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	rec := httptest.NewRecorder()
	Gzip(handler(http.StatusOK)).ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected Content-Encoding: gzip")
	}
	if rec.Header().Get("ETag") != `W/"abc"` {
		t.Errorf("ETag = %s, want weak", rec.Header().Get("ETag"))
	}
	compressed := rec.Body.Len()
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	if string(got) != body {
		t.Error("decompressed body differs")
	}
	t.Logf("%d bytes → %d bytes gzip", len(body), compressed)
}

func TestNoGzip(t *testing.T) {
	// Клиент не умеет gzip
	rec := httptest.NewRecorder()
	Gzip(handler(http.StatusOK)).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != body {
		t.Error("must pass through without gzip")
	}

	// 304 — без тела и без Content-Encoding
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()
	Gzip(handler(http.StatusNotModified)).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified || rec.Header().Get("Content-Encoding") != "" || rec.Body.Len() != 0 {
		t.Errorf("304: code=%d enc=%q len=%d", rec.Code, rec.Header().Get("Content-Encoding"), rec.Body.Len())
	}
}
