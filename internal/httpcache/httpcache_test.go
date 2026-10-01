package httpcache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// body — что отдаёт «handler»; меняем, чтобы проверить смену ETag.
func newServer(body *string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/cities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(*body))
	})
	mux.HandleFunc("GET /v1/cities/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	return Middleware([]Rule{{Prefix: "/v1/cities", MaxAge: 24 * time.Hour}})(mux)
}

func get(h http.Handler, url, ifNoneMatch string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestETagAnd304(t *testing.T) {
	body := `{"items":[]}`
	h := newServer(&body)

	// 1. Первый запрос — 200, ETag, Cache-Control
	first := get(h, "/v1/cities", "")
	etag := first.Header().Get("ETag")
	if first.Code != 200 || etag == "" {
		t.Fatalf("code=%d etag=%q", first.Code, etag)
	}
	if cc := first.Header().Get("Cache-Control"); cc != "public, max-age=86400" {
		t.Errorf("Cache-Control = %q", cc)
	}
	if first.Body.String() != body {
		t.Errorf("тело изменилось: %q", first.Body.String())
	}

	// 2. Повтор с тем же ETag — 304 без тела
	second := get(h, "/v1/cities", etag)
	if second.Code != http.StatusNotModified || second.Body.Len() != 0 {
		t.Errorf("ожидали 304 без тела, получили %d, %d байт", second.Code, second.Body.Len())
	}

	// 3. Слабый ETag и список тоже понимаем
	if c := get(h, "/v1/cities", `"other", W/`+etag).Code; c != http.StatusNotModified {
		t.Errorf("список с W/: code=%d", c)
	}

	// 4. Данные изменились — новый ETag, старый больше не подходит
	body = `{"items":[{"id":"dushanbe"}]}`
	third := get(h, "/v1/cities", etag)
	if third.Code != 200 || third.Header().Get("ETag") == etag {
		t.Errorf("после изменения данных: code=%d etag=%s", third.Code, third.Header().Get("ETag"))
	}
}

func TestErrorsNotCached(t *testing.T) {
	body := ""
	rec := get(newServer(&body), "/v1/cities/atlantis", "")
	if rec.Code != 404 || rec.Header().Get("ETag") != "" || rec.Header().Get("Cache-Control") != "" {
		t.Errorf("ошибку закешировали: code=%d etag=%q cc=%q", rec.Code, rec.Header().Get("ETag"), rec.Header().Get("Cache-Control"))
	}
}

func TestCoordinatesArePrivate(t *testing.T) {
	body := `{}`
	rec := get(newServer(&body), "/v1/cities?lat=38.56&lon=68.79", "")
	if cc := rec.Header().Get("Cache-Control"); cc != "private, max-age=86400" {
		t.Errorf("запрос с координатами: Cache-Control = %q, ожидали private", cc)
	}
}

func TestUnmatchedPathUntouched(t *testing.T) {
	body := ""
	rec := get(newServer(&body), "/health", "")
	if rec.Header().Get("ETag") != "" || rec.Header().Get("Cache-Control") != "" {
		t.Error("/health не должен кешироваться")
	}
}
