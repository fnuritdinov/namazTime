package prayer

import "time"

// Timings — время намазов в формате "HH:MM" (местное время точки).
type Timings struct {
	Fajr    string
	Sunrise string
	Dhuhr   string
	Asr     string
	Maghrib string
	Isha    string
}

// DayTimings — время намаза на конкретный день.
type DayTimings struct {
	Date      time.Time
	HijriDate string
	Source    string // "aladhan" или "shuroiulamo"
	Timings   Timings
}
