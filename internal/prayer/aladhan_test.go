package prayer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const fakeAladhanJSON = `{
  "code": 200,
  "data": {
    "timings": {
      "Fajr": "04:52", "Sunrise": "06:15", "Dhuhr": "12:14",
      "Asr": "15:48 (+05)", "Maghrib": "18:12", "Isha": "19:30"
    },
    "date": { "hijri": { "date": "18-04-1448" } }
  }
}`

func TestAladhanClient_Fetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/timings/30-09-2026" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("latitude") != "41.31" || q.Get("longitude") != "69.24" {
			t.Errorf("coords = %s, %s", q.Get("latitude"), q.Get("longitude"))
		}
		if q.Get("school") != "1" {
			t.Errorf("school = %s", q.Get("school"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAladhanJSON))
	}))
	defer srv.Close()

	c := NewAladhanClient(srv.URL)
	date := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	got, err := c.Fetch(context.Background(), 41.31, 69.24, date, 3, 1)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.Timings.Fajr != "04:52" {
		t.Errorf("Fajr = %q", got.Timings.Fajr)
	}
	if got.Timings.Asr != "15:48" {
		t.Errorf("Asr = %q, суффикс часового пояса должен быть убран", got.Timings.Asr)
	}
	if got.HijriDate != "18-04-1448" {
		t.Errorf("HijriDate = %q", got.HijriDate)
	}
	if got.Source != "aladhan" {
		t.Errorf("Source = %q", got.Source)
	}
}

func TestAladhanClient_Fetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewAladhanClient(srv.URL)
	_, err := c.Fetch(context.Background(), 41.31, 69.24, time.Now(), 3, 1)
	if err == nil {
		t.Fatal("ожидали ошибку при ответе 500")
	}
}
