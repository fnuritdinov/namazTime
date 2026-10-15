package official

import (
	"fmt"
	"time"
)

// Row — один день официальной таблицы: начало и конец каждого намаза ("HH:MM", местное время).
// IshaEnd — утро СЛЕДУЮЩЕГО дня. Восход = FajrEnd.
type Row struct {
	Date                            time.Time
	HijriDay, HijriMonth, HijriYear int

	FajrStart, FajrEnd       string
	DhuhrStart, DhuhrEnd     string
	AsrStart, AsrEnd         string
	MakruhStart, MakruhEnd   string
	MaghribStart, MaghribEnd string
	IshaStart, IshaEnd       string
}

// Validate проверяет строку до записи в базу: формат времени, порядок намазов за день, хиджру.
func (r Row) Validate() error {
	seq := []struct{ name, v string }{
		{"fajr_start", r.FajrStart}, {"fajr_end", r.FajrEnd},
		{"dhuhr_start", r.DhuhrStart}, {"dhuhr_end", r.DhuhrEnd},
		{"asr_start", r.AsrStart}, {"asr_end", r.AsrEnd},
		{"makruh_start", r.MakruhStart}, {"makruh_end", r.MakruhEnd},
		{"maghrib_start", r.MaghribStart}, {"maghrib_end", r.MaghribEnd},
		{"isha_start", r.IshaStart},
	}
	for i, s := range seq {
		if _, err := time.Parse("15:04", s.v); err != nil {
			return fmt.Errorf("%s %q: expected HH:MM", s.name, s.v)
		}
		// "HH:MM" с ведущими нулями можно сравнивать как строки
		if i > 0 && s.v <= seq[i-1].v {
			return fmt.Errorf("%s (%s) must be later than %s (%s)", s.name, s.v, seq[i-1].name, seq[i-1].v)
		}
	}
	if _, err := time.Parse("15:04", r.IshaEnd); err != nil {
		return fmt.Errorf("isha_end %q: expected HH:MM", r.IshaEnd)
	}
	if r.IshaEnd >= "12:00" {
		return fmt.Errorf("isha_end (%s) must be the next morning (before 12:00)", r.IshaEnd)
	}
	if r.HijriDay < 1 || r.HijriDay > 30 || r.HijriMonth < 1 || r.HijriMonth > 12 || r.HijriYear < 1400 {
		return fmt.Errorf("hijri %02d.%02d.%d looks wrong", r.HijriDay, r.HijriMonth, r.HijriYear)
	}
	return nil
}
