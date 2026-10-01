package schedule

import (
	"context"
	"errors"
	"fmt"
	"nTime/internal/official"
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

// OfficialStore — официальные расписания (в main — official.Repository).
type OfficialStore interface {
	// Get возвращает расписание города на диапазон дат или official.ErrNotFound.
	Get(ctx context.Context, cityID string, from, to time.Time) (official.Timetable, error)
}

type Service struct {
	cities   CityGetter
	official OfficialStore
}

func NewService(cities CityGetter, official OfficialStore) *Service {
	return &Service{cities: cities, official: official}
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
	Source     string            // "official" | "calculated"
	SourceName map[string]string // только для official
	UpdatedAt  *time.Time        // только для official
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
	// 4. Источник: official, если есть таблица на ВЕСЬ диапазон; иначе расчёт (для auto)
	switch r.Source {
	case "", "auto", "official":
		sch, err := s.officialSchedule(ctx, out, r, loc)
		if err == nil {
			return sch, nil
		}
		if !errors.Is(err, official.ErrNotFound) {
			return Schedule{}, err
		}
		if r.Source == "official" {
			return Schedule{}, fmt.Errorf("%w: no official timetable for this location and dates", ErrUnsupported)
		}
		out.Source = "calculated" // auto → фолбэк на расчёт
	case "calculated":
		out.Source = "calculated"
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

// officialSchedule собирает расписание из официальной таблицы.
// Возвращает official.ErrNotFound, если таблицы нет или она покрывает не все дни.
func (s *Service) officialSchedule(ctx context.Context, out Schedule, r Request, loc *time.Location) (Schedule, error) {
	if out.CityID == "" {
		return Schedule{}, official.ErrNotFound // по координатам официальных таблиц нет
	}
	tt, err := s.official.Get(ctx, out.CityID, r.From, r.To)
	if err != nil {
		return Schedule{}, err
	}

	out.Source = "official"
	out.SourceName = tt.SourceName
	updated := tt.UpdatedAt
	out.UpdatedAt = &updated
	out.Days = nil

	for d := r.From; !d.After(r.To); d = d.AddDate(0, 0, 1) {
		row, ok := tt.Days[d.Format("2006-01-02")]
		if !ok {
			return Schedule{}, official.ErrNotFound // таблица покрывает не весь диапазон
		}
		day, err := officialDay(d, row, tt.Adjustments, loc)
		if err != nil {
			return Schedule{}, err
		}
		out.Days = append(out.Days, day)
	}
	return out, nil
}
