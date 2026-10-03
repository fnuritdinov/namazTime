package hijri

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestCalendar(t *testing.T) {
	// Шурои уламо: 4-й месяц 1448 начался 13.09.2026, 5-й — 12.10.2026 (в 4-м месяце 29 дней)
	cal := NewCalendar([]MonthStart{
		{Year: 1448, Month: 5, Start: date("2026-10-12")},
		{Year: 1448, Month: 4, Start: date("2026-09-13")},
	})

	tests := []struct {
		date string
		want Date
	}{
		{"2026-09-13", Date{1, 4, 1448}},  // первый день месяца
		{"2026-09-30", Date{18, 4, 1448}}, // как у Шуро (табличный расчёт даёт 17)
		{"2026-10-11", Date{29, 4, 1448}}, // последний день: месяц из 29 дней
		{"2026-10-12", Date{1, 5, 1448}},
		{"2026-11-09", Date{29, 5, 1448}},
		{"2026-11-10", FromGregorian(date("2026-11-10"))}, // 30-й день 5-го месяца: 6-й неизвестен → расчёт
		{"2026-09-01", FromGregorian(date("2026-09-01"))}, // раньше всех данных → расчёт
	}
	for _, tt := range tests {
		if got := cal.Date(date(tt.date)); got != tt.want {
			t.Errorf("%s → %+v, want %+v", tt.date, got, tt.want)
		}
	}
}

func TestCalendar_30Days(t *testing.T) {
	// Рамадан 1448: 08.02.2027, Шавваль — 10.03.2027 → 30 дней
	cal := NewCalendar([]MonthStart{
		{Year: 1448, Month: 9, Start: date("2027-02-08")},
		{Year: 1448, Month: 10, Start: date("2027-03-10")},
	})
	if got := cal.Date(date("2027-03-09")); got != (Date{30, 9, 1448}) {
		t.Errorf("2027-03-09 → %+v, want 30.09.1448", got)
	}
}

func TestCalendar_TimeZone(t *testing.T) {
	// Время суток и часовой пояс не должны сдвигать день
	cal := NewCalendar([]MonthStart{{Year: 1448, Month: 4, Start: date("2026-09-13")}})
	dushanbe, _ := time.LoadLocation("Asia/Dushanbe")
	late := time.Date(2026, 9, 30, 23, 30, 0, 0, dushanbe)
	if got := cal.Date(late); got != (Date{18, 4, 1448}) {
		t.Errorf("got %+v", got)
	}
}

func TestCalendar_Empty(t *testing.T) {
	d := date("2026-09-30")
	if got := NewCalendar(nil).Date(d); got != FromGregorian(d) {
		t.Errorf("empty calendar must fall back to FromGregorian, got %+v", got)
	}
}
