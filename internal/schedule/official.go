package schedule

import (
	"fmt"
	"time"

	"nTime/internal/hijri"
	"nTime/internal/official"
)

const imsakBefore = 10 * time.Minute

// officialDay превращает строку официальной таблицы во время с часовым поясом,
// применяет поправки города (±минуты) и строит диапазоны намазов.
func officialDay(date time.Time, d official.Day, adj map[string]int, loc *time.Location, cal hijri.Calendar) (Day, error) {
	// at: "HH:MM" на дату date (+ addDays) в поясе города, с поправкой города adjKey.
	at := func(name, hhmm, adjKey string, addDays int) (time.Time, error) {
		t, err := time.Parse("15:04", hhmm)
		if err != nil {
			return time.Time{}, fmt.Errorf("official %s %q: %w", name, hhmm, err)
		}
		y, m, dd := date.AddDate(0, 0, addDays).Date()
		res := time.Date(y, m, dd, t.Hour(), t.Minute(), 0, 0, loc)
		return res.Add(time.Duration(adj[adjKey]) * time.Minute), nil
	}

	var out Day
	out.Date = date
	var err error

	// 1. Начала намазов (как было)
	starts := []struct {
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
	for _, f := range starts {
		if *f.dst, err = at(f.name, f.src, f.name, 0); err != nil {
			return Day{}, err
		}
	}

	if d.Imsak != "" {
		if out.Times.Imsak, err = at("imsak", d.Imsak, "imsak", 0); err != nil {
			return Day{}, err
		}
	} else {
		out.Times.Imsak = out.Times.Fajr.Add(-imsakBefore)
	}

	// 2. Диапазоны: сначала по правилу, потом концы из таблицы Шуро, если они есть.
	//    Поправка города для конца — та же, что для близкого по смыслу времени.
	p := periodsByRule(out.Times, out.Times.Fajr.AddDate(0, 0, 1))
	ends := []struct {
		name    string
		src     string
		adjKey  string
		addDays int // isha_end — утро СЛЕДУЮЩЕГО дня
		dst     *time.Time
	}{
		{"fajr_end", d.FajrEnd, "sunrise", 0, &p.Fajr.End},
		{"dhuhr_end", d.DhuhrEnd, "dhuhr", 0, &p.Dhuhr.End},
		{"asr_end", d.AsrEnd, "asr", 0, &p.Asr.End},
		{"maghrib_end", d.MaghribEnd, "maghrib", 0, &p.Maghrib.End},
		{"isha_end", d.IshaEnd, "isha", 1, &p.Isha.End},
	}
	for _, e := range ends {
		if e.src == "" {
			continue // в таблице нет — остаётся конец по правилу
		}
		if *e.dst, err = at(e.name, e.src, e.adjKey, e.addDays); err != nil {
			return Day{}, err
		}
	}

	// Макрух привязан к закату — поправка как у Шома.
	if d.MakruhStart != "" && d.MakruhEnd != "" {
		var mk Range
		if mk.Start, err = at("makruh_start", d.MakruhStart, "maghrib", 0); err != nil {
			return Day{}, err
		}
		if mk.End, err = at("makruh_end", d.MakruhEnd, "maghrib", 0); err != nil {
			return Day{}, err
		}
		p.Makruh = &mk
	}
	out.Periods = p

	// 3. Хиджра из самой таблицы Шуро; если её там нет — из календаря страны
	if d.HijriDay > 0 {
		out.Hijri = hijri.Date{Day: d.HijriDay, Month: d.HijriMonth, Year: d.HijriYear}
	} else {
		out.Hijri = cal.Date(date)
	}
	return out, nil
}
