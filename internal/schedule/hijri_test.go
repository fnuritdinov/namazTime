package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"nTime/internal/hijri"
)

// fakeMonths — начала месяцев Шурои уламо для Таджикистана.
type fakeMonths struct {
	err     error
	country string // какую страну спросили
}

func (f *fakeMonths) Starts(_ context.Context, country string, from, to time.Time) ([]hijri.MonthStart, error) {
	f.country = country
	return []hijri.MonthStart{
		{Year: 1448, Month: 4, Start: day("2026-09-13")},
		{Year: 1448, Month: 5, Start: day("2026-10-12")},
	}, f.err
}

func TestGet_HijriFromCountryCalendar(t *testing.T) {
	months := &fakeMonths{}
	svc := NewService(fakeCities{"dushanbe": dushanbe}, fakeOfficial{}, months)

	got, err := svc.Get(context.Background(), Request{
		CityID: "dushanbe", From: day("2026-09-30"), To: day("2026-10-12"), Source: "calculated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if months.country != "TJ" {
		t.Errorf("спросили страну %q, ожидали TJ", months.country)
	}
	first, last := got.Days[0].Hijri, got.Days[len(got.Days)-1].Hijri
	if first != (hijri.Date{Day: 18, Month: 4, Year: 1448}) { // по формуле было бы 17
		t.Errorf("2026-09-30 → %+v, ожидали 18.04.1448 как у Шуро", first)
	}
	if last != (hijri.Date{Day: 1, Month: 5, Year: 1448}) {
		t.Errorf("2026-10-12 → %+v, ожидали 01.05.1448", last)
	}
}

func TestGet_HijriByCoordinatesIsCalculated(t *testing.T) {
	// По координатам страна неизвестна → только расчёт, база не спрашивается
	months := &fakeMonths{}
	svc := NewService(fakeCities{}, fakeOfficial{}, months)
	lat, lon := 38.56, 68.79
	got, err := svc.Get(context.Background(), Request{
		Lat: &lat, Lon: &lon, TimeZoneID: "Asia/Dushanbe", From: day("2026-09-30"), To: day("2026-09-30"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if months.country != "" {
		t.Errorf("база не должна спрашиваться, спросили %q", months.country)
	}
	if got.Days[0].Hijri != hijri.FromGregorian(day("2026-09-30")) {
		t.Errorf("got %+v", got.Days[0].Hijri)
	}
}

func TestGet_HijriStoreError(t *testing.T) {
	svc := NewService(fakeCities{"dushanbe": dushanbe}, fakeOfficial{}, &fakeMonths{err: errors.New("db down")})
	_, err := svc.Get(context.Background(), Request{CityID: "dushanbe", From: day("2026-09-30"), To: day("2026-09-30")})
	if err == nil {
		t.Error("ожидали ошибку базы")
	}
}
