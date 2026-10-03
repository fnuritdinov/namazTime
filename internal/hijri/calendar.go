package hijri

import (
	"sort"
	"time"
)

// MonthStart — официальное начало месяца хиджры в стране (из таблицы hijri_months).
type MonthStart struct {
	Year, Month int
	Start       time.Time
}

// Calendar переводит даты с учётом официальных начал месяцев.
// Где официальных данных нет — табличный расчёт (FromGregorian).
type Calendar struct {
	starts []MonthStart // по возрастанию Start
}

// NewCalendar — календарь по известным началам месяцев (порядок не важен; nil — только расчёт).
func NewCalendar(starts []MonthStart) Calendar {
	s := make([]MonthStart, len(starts))
	copy(s, starts)
	sort.Slice(s, func(i, j int) bool { return s[i].Start.Before(s[j].Start) })
	return Calendar{starts: s}
}

// Date — дата хиджры для григорианского дня t.
func (c Calendar) Date(t time.Time) Date {
	day := civil(t)

	// Последний месяц, начавшийся не позже t
	i := sort.Search(len(c.starts), func(i int) bool { return civil(c.starts[i].Start).After(day) }) - 1
	if i < 0 {
		return FromGregorian(t)
	}
	m := c.starts[i]
	offset := int(day.Sub(civil(m.Start)).Hours() / 24) // 0 — первый день месяца

	switch {
	case offset <= 28:
		// Дни 1–29 есть в любом месяце
	case offset == 29 && i+1 < len(c.starts) && isNext(m, c.starts[i+1]):
		// 30-й день: следующий месяц известен и начинается позже — значит, в этом месяце 30 дней
	default:
		// 30-й день без данных о следующем месяце или больше 30 — не угадываем
		return FromGregorian(t)
	}
	return Date{Day: offset + 1, Month: m.Month, Year: m.Year}
}

// isNext — b идёт сразу после a (12-й месяц → 1-й месяц следующего года).
func isNext(a, b MonthStart) bool {
	if a.Month == 12 {
		return b.Month == 1 && b.Year == a.Year+1
	}
	return b.Month == a.Month+1 && b.Year == a.Year
}

// civil — полночь UTC того же календарного дня (часовой пояс и время не мешают считать дни).
func civil(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
