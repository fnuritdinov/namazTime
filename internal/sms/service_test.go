package sms

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestCleanName(t *testing.T) {
	ok := map[string]string{"": "", "  Фаррух  ": "Фаррух", "Анна-Мария": "Анна-Мария", "Farrukh N": "Farrukh N"}
	for in, want := range ok {
		got, err := CleanName(in)
		if err != nil || got != want {
			t.Errorf("CleanName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"bit.ly/x", "Звони 901234567", "Ахмад!", strings.Repeat("а", 21)} {
		if _, err := CleanName(bad); err == nil {
			t.Errorf("CleanName(%q): want error", bad)
		}
	}
}

func TestPrayerTextFitsTwoSMS(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 25, h, m, 0, 0, time.UTC) }
	p := Prayers{Date: at(0, 0), Fajr: at(5, 15), Dhuhr: at(12, 20), Asr: at(15, 30), Maghrib: at(17, 45), Isha: at(19, 5)}

	for _, lang := range []string{"tg", "ru"} {
		for _, name := range []string{"", "Фаррух", strings.Repeat("Ж", MaxNameLen)} {
			if n := len([]rune(PrayerText(lang, name, p))); n > MaxTwoSMS {
				t.Errorf("lang=%s name=%q: %d > %d", lang, name, n, MaxTwoSMS)
			}
		}
	}
	want := "Фаррух вақти намозро хотиррасон мекунад (25.10):\nБомдод 05:15\nПешин 12:20\nАср 15:30\nШом 17:45\nХуфтан 19:05"
	if got := PrayerText("tg", "Фаррух", p); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// memStore — журнал в памяти вместо Postgres.
type memStore struct{ rows [][2]string } // [deviceID, phoneHash]

func (m *memStore) CountByDevice(_ context.Context, id string, _ time.Time) (int, error) {
	n := 0
	for _, r := range m.rows {
		if r[0] == id {
			n++
		}
	}
	return n, nil
}
func (m *memStore) CountByPhone(_ context.Context, h string, _ time.Time) (int, error) {
	n := 0
	for _, r := range m.rows {
		if r[1] == h {
			n++
		}
	}
	return n, nil
}
func (m *memStore) CountAll(context.Context, time.Time) (int, error) { return len(m.rows), nil }
func (m *memStore) Log(_ context.Context, id, h, _ string) error {
	m.rows = append(m.rows, [2]string{id, h})
	return nil
}

type okSender struct{ sent int }

func (o *okSender) Send(context.Context, string, string, string) (Result, error) {
	o.sent++
	return Result{Status: "pending"}, nil
}

func TestServiceLimits(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	snd := &okSender{}
	svc := NewService(snd, &memStore{}, Limits{PerDevice: 2, PerPhone: 2, PerDay: 3}, "secret", time.UTC, log)

	if left, err := svc.Send(ctx, "dev_a", "+992 90 123 4567", "x"); err != nil || left != 1 {
		t.Fatalf("1st: left=%d err=%v", left, err)
	}
	if left, err := svc.Send(ctx, "dev_a", "992901111111", "x"); err != nil || left != 0 {
		t.Fatalf("2nd: left=%d err=%v", left, err)
	}
	if _, err := svc.Send(ctx, "dev_a", "992902222222", "x"); !errors.Is(err, ErrDeviceLimit) {
		t.Fatalf("3rd: want ErrDeviceLimit, got %v", err)
	}
	if _, err := svc.Send(ctx, "dev_b", "992901234567", "x"); err != nil {
		t.Fatalf("dev_b 1st: %v", err)
	}
	if _, err := svc.Send(ctx, "dev_c", "992901234567", "x"); !errors.Is(err, ErrPhoneLimit) && !errors.Is(err, ErrGlobalLimit) {
		t.Fatalf("dev_c: want limit error, got %v", err)
	}
	if _, err := svc.Send(ctx, "dev_c", "123", "x"); !errors.Is(err, ErrBadPhone) {
		t.Fatalf("bad phone: got %v", err)
	}
	if snd.sent != 3 {
		t.Fatalf("sent=%d, want 3", snd.sent)
	}
}
