package schedule

import (
	"fmt"
	"time"

	"nTime/internal/hijri"
	"nTime/internal/official"
)

const imsakBefore = 10 * time.Minute

// officialDay превращает строку официальной таблицы во время с часовым поясом
// и применяет поправки города (±минуты).
func officialDay(date time.Time, d official.Day, adj map[string]int, loc *time.Location, cal hijri.Calendar) (Day, error) {
	at := func(name, hhmm string) (time.Time, error) {
		t, err := time.Parse("15:04", hhmm)
		if err != nil {
			return time.Time{}, fmt.Errorf("official %s %q: %w", name, hhmm, err)
		}
		y, m, dd := date.Date()
		res := time.Date(y, m, dd, t.Hour(), t.Minute(), 0, 0, loc)
		return res.Add(time.Duration(adj[name]) * time.Minute), nil
	}

	var out Day
	out.Date = date
	var err error
	fields := []struct {
		name string
		src  string
		dst  *time.Time
	}{
		{"fajr", d.Fajr, &out.Times.Fajr},
		{"sunrise", d.Sunrise, &out.Times.Sunrise},
		{"dhuhr", d.Dhuhr, &out.Times.Dhuhr},
		{"asr", d.Asr, &out.Times.Asr},
		{"maghrib", d.Maghrib, &out.Times.Maghrib},
		{"isha", d.Isha, &out.Times.Isha},
	}
	for _, f := range fields {
		if *f.dst, err = at(f.name, f.src); err != nil {
			return Day{}, err
		}
	}

	if d.Imsak != "" {
		if out.Times.Imsak, err = at("imsak", d.Imsak); err != nil {
			return Day{}, err
		}
	} else {
		out.Times.Imsak = out.Times.Fajr.Add(-imsakBefore)
	}

	// Хиджра из самой таблицы Шуро; если её там нет — из календаря страны
	if d.HijriDay > 0 {
		out.Hijri = hijri.Date{Day: d.HijriDay, Month: d.HijriMonth, Year: d.HijriYear}
	} else {
		out.Hijri = cal.Date(date)
	}
	return out, nil
}
