package schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nTime/internal/city"
	"nTime/internal/hijri"
	"nTime/internal/prayertime"
)

const maxDays = 62 // §6 ТЗ: диапазон не больше 62 дней

// InvalidError — неверный параметр запроса (→ 400 invalid_argument).
type InvalidError struct{ Msg string }

func (e *InvalidError) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &InvalidError{Msg: fmt.Sprintf(format, args...)}
}

// ErrUnsupported — запрос корректный, но выполнить нельзя (→ 422 unsupported).
var ErrUnsupported = errors.New("unsupported")

// CityGetter — откуда берём город (в main — city.Repository, в тестах — заглушка).
type CityGetter interface {
	Get(ctx context.Context, id string) (city.City, error)
}

type Service struct {
	cities CityGetter
}

func NewService(cities CityGetter) *Service {
	return &Service{cities: cities}
}

// Request — параметры GET /v1/prayer-times.
type Request struct {
	CityID     string   // либо город…
	Lat, Lon   *float64 // …либо координаты + часовой пояс
	TimeZoneID string
	From, To   time.Time
	Method     string // "" — suggestedMethod города
	Madhab     string // "" — suggestedMadhab города
	Source     string // "", "auto", "official", "calculated"
}

type Day struct {
	Date  time.Time
	Hijri hijri.Date
	Times prayertime.Times
}

type Schedule struct {
	CityID     string // "" — если запрос по координатам
	TimeZoneID string
	Method     string
	Madhab     string
	Source     string // "official" | "calculated"
	Days       []Day
}

func (s *Service) Get(ctx context.Context, r Request) (Schedule, error) {
	// 1. Диапазон дат
	if r.To.Before(r.From) {
		return Schedule{}, invalid("to must not be before from")
	}
	if n := int(r.To.Sub(r.From).Hours()/24) + 1; n > maxDays {
		return Schedule{}, invalid("date range must not exceed %d days, got %d", maxDays, n)
	}

	// 2. Где считаем: город или координаты
	out := Schedule{Method: r.Method, Madhab: r.Madhab}
	var lat, lon float64

	switch {
	case r.CityID != "":
		c, err := s.cities.Get(ctx, r.CityID)
		if err != nil {
			return Schedule{}, err // city.ErrNotFound → 404
		}
		lat, lon = c.Lat, c.Lon
		out.CityID, out.TimeZoneID = c.ID, c.TimeZone
		if out.Method == "" {
			out.Method = c.SuggestedMethod
		}
		if out.Madhab == "" {
			out.Madhab = c.SuggestedMadhab
		}

	case r.Lat != nil && r.Lon != nil && r.TimeZoneID != "":
		lat, lon = *r.Lat, *r.Lon
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return Schedule{}, invalid("lat must be in [-90, 90], lon in [-180, 180]")
		}
		out.TimeZoneID = r.TimeZoneID
		if out.Method == "" {
			out.Method = "muslimWorldLeague"
		}
		if out.Madhab == "" {
			out.Madhab = prayertime.Hanafi
		}

	default:
		return Schedule{}, invalid("either cityId or lat, lon and timeZoneId are required")
	}

	// 3. Проверяем параметры расчёта
	loc, err := time.LoadLocation(out.TimeZoneID)
	if err != nil || out.TimeZoneID == "Local" {
		return Schedule{}, invalid("unknown timeZoneId %q", out.TimeZoneID)
	}
	method, ok := prayertime.MethodByID(out.Method)
	if !ok {
		return Schedule{}, invalid("unknown method %q", out.Method)
	}
	if out.Madhab != prayertime.Hanafi && out.Madhab != prayertime.Shafii {
		return Schedule{}, invalid("madhab must be hanafi or shafii")
	}

	// 4. Источник. Официальные таблицы появятся в D.3 — пока всегда расчёт.
	switch r.Source {
	case "", "auto", "calculated":
		out.Source = "calculated"
	case "official":
		return Schedule{}, fmt.Errorf("%w: no official timetable for this location", ErrUnsupported)
	default:
		return Schedule{}, invalid("source must be auto, official or calculated")
	}

	// 5. Считаем каждый день
	out.Days = make([]Day, 0, int(r.To.Sub(r.From).Hours()/24)+1)
	for d := r.From; !d.After(r.To); d = d.AddDate(0, 0, 1) {
		t, err := prayertime.Calculate(d, lat, lon, loc, method, out.Madhab)
		if errors.Is(err, prayertime.ErrPolar) {
			return Schedule{}, fmt.Errorf("%w: %v", ErrUnsupported, err)
		}
		if err != nil {
			return Schedule{}, err
		}
		out.Days = append(out.Days, Day{Date: d, Hijri: hijri.FromGregorian(d), Times: t})
	}
	return out, nil
}
