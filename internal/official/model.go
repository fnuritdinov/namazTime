package official

import (
	"errors"
	"time"
)

// ErrNotFound — у города нет официального расписания.
var ErrNotFound = errors.New("no official timetable")

// Day — один день официального расписания. Время — местное "HH:MM".
// Поля *End и MakruhStart/MakruhEnd — "", если в таблице их нет (старый импорт).
type Day struct {
	Fajr, Sunrise, Dhuhr, Asr, Maghrib, Isha string // начала намазов (Sunrise = конец Бомдода)
	Imsak                                    string // "" — источник не публикует

	FajrEnd, DhuhrEnd, AsrEnd, MaghribEnd string
	IshaEnd                               string // время СЛЕДУЮЩЕГО утра
	MakruhStart, MakruhEnd                string

	HijriDay, HijriMonth, HijriYear int // 0 — нет официальной даты хиджры
}

// Timetable — официальное расписание города на диапазон дат.
type Timetable struct {
	SourceName  map[string]string // {"tg": "Шӯрои уламои Тоҷикистон", ...}
	UpdatedAt   time.Time
	Days        map[string]Day // ключ — дата "2006-01-02"
	Adjustments map[string]int // поправки города в минутах: "fajr" → -3
}
