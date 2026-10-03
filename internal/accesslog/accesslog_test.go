package accesslog

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMaskQuery(t *testing.T) {
	q, _ := url.ParseQuery("lat=38.5598&lon=68.787&timeZoneId=Asia/Dushanbe")
	got := MaskQuery(q)
	if strings.Contains(got, "38.5598") || strings.Contains(got, "68.787") {
		t.Errorf("coordinates leaked: %s", got)
	}
	if got != "lat=***&lon=***&timeZoneId=Asia/Dushanbe" {
		t.Errorf("got %s", got)
	}
}

func TestMiddlewareLogsWithoutCoordinates(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Middleware(log, func(context.Context) string { return "req-1" })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("nope"))
		}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/cities/nearest?lat=38.56&lon=68.79", nil))

	line := buf.String()
	if strings.Contains(line, "38.56") || strings.Contains(line, "68.79") {
		t.Errorf("coordinates in log: %s", line)
	}
	for _, want := range []string{`"status":404`, `"bytes":4`, `"requestId":"req-1"`, `"path":"/v1/cities/nearest"`} {
		if !strings.Contains(line, want) {
			t.Errorf("log %s does not contain %s", line, want)
		}
	}
}
