package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"nTime/internal/official"
)

// fakeOfficial — официальное расписание Душанбе на 1–2 октября 2026 (данные Шурои уламо).
type fakeOfficial struct {
	adjustments map[string]map[string]int // cityID → поправки
}

func (f fakeOfficial) Get(_ context.Context, cityID string, from, to time.Time) (official.Timetable, error) {
	if cityID != "dushanbe" && cityID != "khujand" {
		return official.Timetable{}, official.ErrNotFound
	}
	return official.Timetable{
		SourceName: map[string]string{"tg": "Шӯрои уламои Тоҷикистон"},
		UpdatedAt:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Days: map[string]official.Day{
			"2026-10-01": {Fajr: "04:51", Sunrise: "06:17", Dhuhr: "12:40", Asr: "16:25", Maghrib: "18:19", Isha: "19:49",
				HijriDay: 19, HijriMonth: 4, HijriYear: 1448},
			"2026-10-02": {Fajr: "04:52", Sunrise: "06:18", Dhuhr: "12:40", Asr: "16:24", Maghrib: "18:18", Isha: "19:48",
				HijriDay: 20, HijriMonth: 4, HijriYear: 1448},
		},
		Adjustments: f.adjustments[cityID],
	}, nil
}

func newOfficialSvc() *Service {
	khujand := dushanbe
	khujand.ID, khujand.Lat, khujand.Lon = "khujand", 40.2826, 69.6222
	return NewService(
		fakeCities{"dushanbe": dushanbe, "khujand": khujand},
		fakeOfficial{adjustments: map[string]map[string]int{
			"khujand": {"fajr": -3, "sunrise": -3, "dhuhr": -3, "asr": -3, "maghrib": -3, "isha": -3},
		}},
	)
}

func TestOfficial_Auto(t *testing.T) {
	got, err := newOfficialSvc().Get(context.Background(), Request{
		CityID: "dushanbe", From: day("2026-10-01"), To: day("2026-10-02"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "official" || got.SourceName["tg"] == "" || got.UpdatedAt == nil {
		t.Fatalf("source=%s name=%v updated=%v", got.Source, got.SourceName, got.UpdatedAt)
	}
	d := got.Days[0]
	if s := d.Times.Dhuhr.Format(time.RFC3339); s != "2026-10-01T12:40:00+05:00" {
		t.Errorf("dhuhr = %s, ожидали официальные 12:40", s)
	}
	if s := d.Times.Imsak.Format("15:04"); s != "04:41" {
		t.Errorf("imsak = %s, ожидали Фаджр − 10 мин", s)
	}
	if d.Hijri.Day != 19 || d.Hijri.Month != 4 {
		t.Errorf("hijri = %+v, ожидали официальную 19.04", d.Hijri)
	}
}

func TestOfficial_Adjustments(t *testing.T) {
	got, err := newOfficialSvc().Get(context.Background(), Request{
		CityID: "khujand", From: day("2026-10-01"), To: day("2026-10-01"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if s := got.Days[0].Times.Fajr.Format("15:04"); s != "04:48" {
		t.Errorf("Худжанд fajr = %s, ожидали 04:51 − 3 = 04:48", s)
	}
}

func TestOfficial_FallbackAndErrors(t *testing.T) {
	svc := newOfficialSvc()
	ctx := context.Background()

	// auto + диапазон шире таблицы → расчёт
	got, err := svc.Get(ctx, Request{CityID: "dushanbe", From: day("2026-10-01"), To: day("2026-10-05")})
	if err != nil || got.Source != "calculated" {
		t.Errorf("auto вне таблицы: source=%s err=%v", got.Source, err)
	}

	// official + диапазон шире таблицы → 422
	_, err = svc.Get(ctx, Request{CityID: "dushanbe", Source: "official", From: day("2026-10-01"), To: day("2026-10-05")})
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("ожидали ErrUnsupported, получили %v", err)
	}

	// calculated — даже если таблица есть
	got, err = svc.Get(ctx, Request{CityID: "dushanbe", Source: "calculated", From: day("2026-10-01"), To: day("2026-10-01")})
	if err != nil || got.Source != "calculated" || got.Days[0].Times.Dhuhr.Format("15:04") != "12:17" {
		t.Errorf("calculated: source=%s err=%v", got.Source, err)
	}

	// по координатам официальных таблиц нет: official → 422
	lat, lon := 38.56, 68.79
	_, err = svc.Get(ctx, Request{Lat: &lat, Lon: &lon, TimeZoneID: "Asia/Dushanbe", Source: "official",
		From: day("2026-10-01"), To: day("2026-10-01")})
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("координаты + official: ожидали ErrUnsupported, получили %v", err)
	}
}
