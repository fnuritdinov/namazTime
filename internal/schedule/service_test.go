package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"nTime/internal/city"
)

type fakeCities map[string]city.City

func (f fakeCities) Get(_ context.Context, id string) (city.City, error) {
	c, ok := f[id]
	if !ok {
		return city.City{}, city.ErrNotFound
	}
	return c, nil
}

var dushanbe = city.City{
	ID: "dushanbe", Country: "TJ", Lat: 38.5598, Lon: 68.787, TimeZone: "Asia/Dushanbe",
	SuggestedMethod: "muslimWorldLeague", SuggestedMadhab: "hanafi",
}

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func newSvc() *Service {
	return NewService(fakeCities{"dushanbe": dushanbe}, fakeOfficial{}, nil)
}

func TestGet_ByCity(t *testing.T) {
	got, err := newSvc().Get(context.Background(), Request{
		CityID: "dushanbe", From: day("2026-09-30"), To: day("2026-10-06"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Days) != 7 {
		t.Fatalf("дней %d, ожидали 7", len(got.Days))
	}
	if got.Method != "muslimWorldLeague" || got.Madhab != "hanafi" || got.Source != "calculated" {
		t.Errorf("method=%s madhab=%s source=%s", got.Method, got.Madhab, got.Source)
	}
	// Эталон ТЗ: Фаджр 30.09 в 04:51 +05:00
	if f := got.Days[0].Times.Fajr.Format(time.RFC3339); f != "2026-09-30T04:51:00+05:00" {
		t.Errorf("fajr = %s", f)
	}
}

func TestGet_ByCoordinates(t *testing.T) {
	lat, lon := 38.56, 68.79
	got, err := newSvc().Get(context.Background(), Request{
		Lat: &lat, Lon: &lon, TimeZoneID: "Asia/Dushanbe",
		From: day("2026-09-30"), To: day("2026-09-30"), Madhab: "shafii",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.CityID != "" || got.Madhab != "shafii" || len(got.Days) != 1 {
		t.Errorf("%+v", got)
	}
}

func TestGet_Errors(t *testing.T) {
	ctx := context.Background()
	svc := newSvc()
	lat, lon := 38.56, 68.79

	invalidCases := map[string]Request{
		"to < from":         {CityID: "dushanbe", From: day("2026-10-02"), To: day("2026-10-01")},
		"больше 62 дней":    {CityID: "dushanbe", From: day("2026-01-01"), To: day("2026-03-05")},
		"нет города":        {From: day("2026-10-01"), To: day("2026-10-01")},
		"нет timeZoneId":    {Lat: &lat, Lon: &lon, From: day("2026-10-01"), To: day("2026-10-01")},
		"плохой timeZoneId": {Lat: &lat, Lon: &lon, TimeZoneID: "Mars/Olympus", From: day("2026-10-01"), To: day("2026-10-01")},
		"плохой method":     {CityID: "dushanbe", Method: "moon", From: day("2026-10-01"), To: day("2026-10-01")},
		"плохой madhab":     {CityID: "dushanbe", Madhab: "maliki", From: day("2026-10-01"), To: day("2026-10-01")},
		"плохой source":     {CityID: "dushanbe", Source: "magic", From: day("2026-10-01"), To: day("2026-10-01")},
	}
	for name, req := range invalidCases {
		var ie *InvalidError
		if _, err := svc.Get(ctx, req); !errors.As(err, &ie) {
			t.Errorf("%s: ожидали InvalidError, получили %v", name, err)
		}
	}

	// ровно 62 дня — можно
	if _, err := svc.Get(ctx, Request{CityID: "dushanbe", From: day("2026-01-01"), To: day("2026-03-03")}); err != nil {
		t.Errorf("62 дня должны быть разрешены: %v", err)
	}

	if _, err := svc.Get(ctx, Request{CityID: "atlantis", From: day("2026-10-01"), To: day("2026-10-01")}); !errors.Is(err, city.ErrNotFound) {
		t.Errorf("ожидали city.ErrNotFound, получили %v", err)
	}

	if _, err := svc.Get(ctx, Request{CityID: "dushanbe", Source: "official", From: day("2026-11-01"), To: day("2026-11-01")}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("ожидали ErrUnsupported, получили %v", err)
	}
}
