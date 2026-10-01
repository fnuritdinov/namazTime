package hijri

import (
	"testing"
	"time"
)

func TestFromGregorian(t *testing.T) {
	tests := []struct {
		date string
		want Date
	}{
		{"2027-02-08", Date{Day: 1, Month: 9, Year: 1448}},  // 1 Рамадана — совпадает с ТЗ (§7)
		{"2027-03-10", Date{Day: 1, Month: 10, Year: 1448}}, // Ид аль-Фитр — совпадает с ТЗ
		{"2026-09-30", Date{Day: 17, Month: 4, Year: 1448}}, // ТЗ/Шуро: 18 — разница в 1 день
		{"2000-01-01", Date{Day: 24, Month: 9, Year: 1420}},
	}
	for _, tt := range tests {
		d, _ := time.Parse("2006-01-02", tt.date)
		if got := FromGregorian(d); got != tt.want {
			t.Errorf("%s → %+v, want %+v", tt.date, got, tt.want)
		}
	}
}

func TestMonthName(t *testing.T) {
	if got := (Date{Month: 9}).MonthName()["en"]; got != "Ramadan" {
		t.Errorf("month 9 = %q", got)
	}
	if (Date{Month: 13}).MonthName() != nil {
		t.Error("месяц 13 не должен существовать")
	}
}
