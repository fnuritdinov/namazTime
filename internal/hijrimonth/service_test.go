package hijrimonth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	months map[string][]Month // "TJ-1448" → месяцы
}

func (f fakeStore) Months(_ context.Context, country string, year int) ([]Month, error) {
	return f.months[country+"-"+itoa(year)], nil
}

func (f fakeStore) Duas(_ context.Context, category string) ([]Dua, error) {
	return []Dua{{ID: "iftar", Arabic: "…", Translations: map[string]string{"ru": "…"}}}, nil
}

func itoa(n int) string { return time.Date(n, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006") }

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestRamadan_ExpectedFromSpec(t *testing.T) {
	// Пример из ТЗ §7: 1448, TJ, начало 2027-02-08, Ид 2027-03-10, длина 30, Лайлат аль-Кадр 2027-03-05
	svc := NewService(fakeStore{months: map[string][]Month{
		"TJ-1448": {
			{Month: 9, Start: day("2027-02-08"), Status: "expected"},
			{Month: 10, Start: day("2027-03-10"), Status: "expected"},
		},
	}})

	r, err := svc.Ramadan(context.Background(), "TJ", 1448)
	if err != nil {
		t.Fatal(err)
	}
	check := map[string]string{
		"start":        r.Start.Format("2006-01-02"),
		"end":          r.End.Format("2006-01-02"),
		"eid":          r.EidAlFitr.Format("2006-01-02"),
		"laylatAlQadr": r.LaylatAlQadr.Format("2006-01-02"),
	}
	want := map[string]string{"start": "2027-02-08", "end": "2027-03-09", "eid": "2027-03-10", "laylatAlQadr": "2027-03-05"}
	for k := range want {
		if check[k] != want[k] {
			t.Errorf("%s = %s, want %s", k, check[k], want[k])
		}
	}
	if r.Length != 30 || r.Status != "expected" || len(r.Duas) != 1 {
		t.Errorf("length=%d status=%s duas=%d", r.Length, r.Status, len(r.Duas))
	}
}

func TestRamadan_29Days(t *testing.T) {
	svc := NewService(fakeStore{months: map[string][]Month{
		"TJ-1449": {
			{Month: 9, Start: day("2028-01-28"), Status: "confirmed"},
			{Month: 10, Start: day("2028-02-26"), Status: "confirmed"},
		},
	}})
	r, err := svc.Ramadan(context.Background(), "TJ", 1449)
	if err != nil {
		t.Fatal(err)
	}
	if r.Length != 29 || r.End.Format("2006-01-02") != "2028-02-25" {
		t.Errorf("length=%d end=%s", r.Length, r.End.Format("2006-01-02"))
	}
}

func TestRamadan_OnlyStartKnown(t *testing.T) {
	// Шавваль ещё не объявлен — ожидаем 30 дней
	svc := NewService(fakeStore{months: map[string][]Month{
		"TJ-1448": {{Month: 9, Start: day("2027-02-08"), Status: "expected"}},
	}})
	r, err := svc.Ramadan(context.Background(), "TJ", 1448)
	if err != nil {
		t.Fatal(err)
	}
	if r.Length != 30 || r.EidAlFitr.Format("2006-01-02") != "2027-03-10" {
		t.Errorf("length=%d eid=%s", r.Length, r.EidAlFitr.Format("2006-01-02"))
	}
}

func TestRamadan_Errors(t *testing.T) {
	svc := NewService(fakeStore{months: map[string][]Month{
		"TJ-1448": {{Month: 4, Start: day("2026-09-13"), Status: "confirmed"}}, // Рамадана ещё нет
	}})
	ctx := context.Background()

	if _, err := svc.Ramadan(ctx, "TJ", 1448); !errors.Is(err, ErrNotFound) {
		t.Errorf("нет 9-го месяца: ожидали ErrNotFound, получили %v", err)
	}
	var ie *InvalidError
	for _, c := range []struct {
		country string
		year    int
	}{{"tj", 1448}, {"TJK", 1448}, {"TJ", 2026}} {
		if _, err := svc.Ramadan(ctx, c.country, c.year); !errors.As(err, &ie) {
			t.Errorf("%s/%d: ожидали InvalidError, получили %v", c.country, c.year, err)
		}
	}
}
