package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"992901234567":      "992901234567",
		"+992 90 123-45-67": "992901234567",
		"(992) 901234567":   "992901234567",
	}
	for in, want := range ok {
		if got, err := NormalizePhone(in); err != nil || got != want {
			t.Errorf("%q: got %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"901234567", "998901234567", "+7 999 123 45 67", "99290123456a", ""} {
		if _, err := NormalizePhone(in); err == nil {
			t.Errorf("%q: want error", in)
		}
	}
}

func TestSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/sms" || r.Header.Get("X-Api-Key") != "k" || r.Header.Get("Idempotency-Key") != "idem" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["phone"] != "992901234567" || body["channel"] != "sms" || body["senderId"] != "Zikr" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc","status":"pending"}`))
	}))
	defer srv.Close()

	c := &Client{http: &http.Client{Timeout: time.Second}, base: srv.URL, apiKey: "k", senderID: "Zikr"}
	r, err := c.Send(context.Background(), "992901234567", "test", "idem")
	if err != nil || string(r.ID) != "abc" || r.Status != "pending" {
		t.Fatalf("got %+v, %v", r, err)
	}

	c.apiKey = "wrong"
	if _, err := c.Send(context.Background(), "992901234567", "test", "idem"); err == nil {
		t.Fatal("want error on 401")
	}
}
