package schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nTime/internal/aladhan"
	"nTime/internal/city"
	"nTime/internal/hijri"
	"nTime/internal/official"
	"nTime/internal/prayertime"
)

const maxDays = 62 // §6 ТЗ: диапазон не больше 62 дней

// defaultMethod — метод для всех, кроме Таджикистана, если приложение не прислало свой.
const defaultMethod = "karachi"

// InvalidError — неверный параметр запроса (→ 400 invalid_argument).
type InvalidError struct{ Msg string }

func (e *InvalidError) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &InvalidError{Msg: fmt.Sprintf(format, args...)}
}

// ErrUnsupported — запрос корректный, но выполнить нельзя (→ 422 unsupported).
var ErrUnsupported = errors.New("unsupported")

// ErrUnavailable — внешний источник времени (Aladhan) недоступен, а в кеше нет (→ 503).
var ErrUnavailable = aladhan.ErrUnavailable

// CityGetter — откуда берём город (в main — city.Repository, в тестах — заглушка).
type CityGetter interface {
	Get(ctx context.Context, id string) (city.City, error)
}

// OfficialStore — официальные расписания (в main — official.Repository).
type OfficialStore interface {
	// Get возвращает расписание города на диапазон дат или official.ErrNotFound.
	Get(ctx context.Context, cityID string, from, to time.Time) (official.Timetable, error)
}

// HijriStore — официальные начала месяцев хиджры (в main — hijrimonth.Repository).
type HijriStore interface {
	// Starts возвращает начала месяцев страны, начавшихся в диапазоне [from, to].
	Starts(ctx context.Context, country string, from, to time.Time) ([]hijri.MonthStart, error)
}

// TimesSource — время намазов из Aladhan (в main — *aladhan.Service).
type TimesSource interface {
	// Month возвращает все дни месяца. cityID == "" — запрос по координатам.
	Month(ctx context.Context, cityID string, m aladhan.Month) ([]aladhan.Day, error)
}

type Service struct {
	cities   CityGetter
	official OfficialStore
	months   HijriStore // nil — дата хиджры только по расчёту
	times    TimesSource
}

func NewService(cities CityGetter, official OfficialStore, months HijriStore, times TimesSource) *Service {
	return &Service{cities: cities, official: official, months: months, times: times}
}

// Request — параметры GET /v1/prayer-times.
type Request struct {
	CityID     string   // либо город…
	Lat, Lon   *float64 // …либо координаты + часовой пояс
	TimeZoneID string
	From, To   time.Time
	Method     string // "" — karachi
	Madhab     string // "" — suggestedMadhab города
	Source     string // "", "auto", "official", "calculated"
}

type Day struct {
	Date    time.Time
	Hijri   hijri.Date
	Times   prayertime.Times
	Periods Periods
	Extra   *Extra // только для времени из Aladhan (не Таджикистан); nil — полей нет
}

// Extra — дополнительные времена от Aladhan.
// Midnight, FirstThird, LastThird могут быть уже следующим днём.
type Extra struct {
	Sunset, Midnight, FirstThird, LastThird time.Time
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
	if out.Method == "" {
		out.Method = defaultMethod
	}
	var lat, lon float64
	var country string // "" — по координатам страна неизвестна
	var hasOfficial bool

	switch {
	case r.CityID != "":
		c, err := s.cities.Get(ctx, r.CityID)
		if err != nil {
			return Schedule{}, err // city.ErrNotFound → 404
		}
		lat, lon, country = c.Lat, c.Lon, c.Country
		hasOfficial = c.HasOfficialTimetable
		out.CityID, out.TimeZoneID = c.ID, c.TimeZone
		if out.Madhab == "" {
			out.Madhab = c.SuggestedMadhab
		}

	case r.Lat != nil && r.Lon != nil && r.TimeZoneID != "":
		lat, lon = *r.Lat, *r.Lon
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return Schedule{}, invalid("lat must be in [-90, 90], lon in [-180, 180]")
		}
		out.TimeZoneID = r.TimeZoneID
		if out.Madhab == "" {
			out.Madhab = prayertime.Hanafi
		}

	default:
		return Schedule{}, invalid("either cityId or lat, lon and timeZoneId are required")
	}

	// 3. Проверяем параметры
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

	// 4. Календарь хиджры страны (официальные начала месяцев)
	cal, err := s.calendar(ctx, country, r.From, r.To)
	if err != nil {
		return Schedule{}, err
	}

	// 5. Источник: official, если есть таблица на ВЕСЬ диапазон.
	// Города с официальной таблицей (TJ): таблица не покрывает период → 422.
	// Остальные города и координаты: auto → Aladhan.
	switch r.Source {
	case "", "auto", "official":
		sch, err := s.officialSchedule(ctx, out, r, loc, cal)
		if err == nil {
			return sch, nil
		}
		if !errors.Is(err, official.ErrNotFound) {
			return Schedule{}, err
		}

		// Для городов Таджикистана не подменяем время Шуро молча.
		// Aladhan — только если приложение само попросило source=calculated.
		if r.Source == "official" || hasOfficial {
			return Schedule{}, fmt.Errorf(
				"%w: official timetable for %s does not cover %s..%s; request a range inside the published month or use source=calculated",
				ErrUnsupported, out.CityID, r.From.Format("2006-01-02"), r.To.Format("2006-01-02"))
		}
		out.Source = "calculated"
	case "calculated":
		out.Source = "calculated"
	default:
		return Schedule{}, invalid("source must be auto, official or calculated")
	}

	// 6. Время из Aladhan — сами не считаем.
	// Нужен ещё 1 день после to: Хуфтан длится до Бомдода следующего дня.
	school := 1 // ханафи
	if out.Madhab == prayertime.Shafii {
		school = 0
	}
	n := int(r.To.Sub(r.From).Hours()/24) + 1
	last := r.From.AddDate(0, 0, n) // этот лишний день нужен только для конца Хуфтана

	byDate := map[string]aladhan.Day{}
	first := time.Date(r.From.Year(), r.From.Month(), 1, 0, 0, 0, 0, r.From.Location())
	for m := first; !m.After(last); m = m.AddDate(0, 1, 0) {
		days, err := s.times.Month(ctx, out.CityID, aladhan.Month{
			Year: m.Year(), Month: m.Month(),
			Lat: lat, Lon: lon, Loc: loc,
			Method: method.AladhanID, School: school,
		})
		if err != nil {
			return Schedule{}, err // aladhan.ErrUnavailable → 503
		}
		for _, d := range days {
			byDate[d.Date.Format("2006-01-02")] = d
		}
	}

	times := make([]prayertime.Times, 0, n+1)
	for i := 0; i <= n; i++ {
		key := r.From.AddDate(0, 0, i).Format("2006-01-02")
		d, ok := byDate[key]
		if !ok {
			return Schedule{}, fmt.Errorf("%w: no data for %s", ErrUnavailable, key)
		}
		times = append(times, prayertime.Times{
			Imsak: d.Imsak, Fajr: d.Fajr, Sunrise: d.Sunrise, Dhuhr: d.Dhuhr,
			Asr: d.Asr, Maghrib: d.Maghrib, Isha: d.Isha,
		})
	}

	out.Days = make([]Day, 0, n)
	for i := 0; i < n; i++ {
		d := r.From.AddDate(0, 0, i)
		day := Day{
			Date:    d,
			Hijri:   cal.Date(d),
			Times:   times[i],
			Periods: periodsByRule(times[i], times[i+1].Fajr),
		}
		if a := byDate[d.Format("2006-01-02")]; !a.Sunset.IsZero() {
			day.Extra = &Extra{Sunset: a.Sunset, Midnight: a.Midnight, FirstThird: a.FirstThird, LastThird: a.LastThird}
		}
		out.Days = append(out.Days, day)
	}
	return out, nil
}

// officialSchedule собирает расписание из официальной таблицы.
// Возвращает official.ErrNotFound, если таблицы нет или она покрывает не все дни.
func (s *Service) officialSchedule(ctx context.Context, out Schedule, r Request, loc *time.Location, cal hijri.Calendar) (Schedule, error) {
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
		day, err := officialDay(d, row, tt.Adjustments, loc, cal)
		if err != nil {
			return Schedule{}, err
		}
		out.Days = append(out.Days, day)
	}
	return out, nil
}

// calendar загружает официальные начала месяцев, нужные для дат [from, to].
// Берём с запасом 30 дней назад (месяц мог начаться раньше from)
// и 1 день вперёд (нужно знать, есть ли в последнем месяце 30-й день).
func (s *Service) calendar(ctx context.Context, country string, from, to time.Time) (hijri.Calendar, error) {
	if country == "" || s.months == nil {
		return hijri.NewCalendar(nil), nil
	}
	starts, err := s.months.Starts(ctx, country, from.AddDate(0, 0, -30), to.AddDate(0, 0, 1))
	if err != nil {
		return hijri.Calendar{}, fmt.Errorf("hijri months: %w", err)
	}
	return hijri.NewCalendar(starts), nil
}
