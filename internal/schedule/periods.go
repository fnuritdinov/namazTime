package schedule

import (
	"time"

	"nTime/internal/prayertime"
)

// Range — время намаза: от Start до End включительно (с точностью до минуты).
type Range struct {
	Start, End time.Time
}

// Periods — начало и конец каждого намаза за день.
type Periods struct {
	Fajr, Dhuhr, Asr, Maghrib, Isha Range
	Makruh                          *Range // только в официальной таблице (Шуро); nil — нет
}

// periodsByRule строит диапазоны, когда концы намазов неизвестны (расчёт по формулам
// или старая официальная таблица без концов):
//   - Бомдод — до восхода;
//   - Пешин, Аср, Шом — до начала следующего намаза − 1 мин;
//   - Хуфтан — до Бомдода следующего дня − 1 мин.
func periodsByRule(t prayertime.Times, nextFajr time.Time) Periods {
	return Periods{
		Fajr:    Range{t.Fajr, t.Sunrise},
		Dhuhr:   Range{t.Dhuhr, t.Asr.Add(-time.Minute)},
		Asr:     Range{t.Asr, t.Maghrib.Add(-time.Minute)},
		Maghrib: Range{t.Maghrib, t.Isha.Add(-time.Minute)},
		Isha:    Range{t.Isha, nextFajr.Add(-time.Minute)},
	}
}
