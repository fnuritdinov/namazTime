package schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nTime/internal/aladhan"
	"nTime/internal/prayertime"
)

// fakeTimes — заглушка Aladhan для тестов: считает время нашей старой формулой,
// чтобы тесты не ходили в интернет.
type fakeTimes struct{}

func (fakeTimes) Month(_ context.Context, _ string, m aladhan.Month) ([]aladhan.Day, error) {
	var method prayertime.Method
	for _, x := range prayertime.Methods {
		if x.AladhanID == m.Method {
			method = x
		}
	}
	madhab := prayertime.Hanafi
	if m.School == 0 {
		madhab = prayertime.Shafii
	}

	var days []aladhan.Day
	for d := time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, time.UTC); d.Month() == m.Month; d = d.AddDate(0, 0, 1) {
		t, err := prayertime.Calculate(d, m.Lat, m.Lon, m.Loc, method, madhab)
		if errors.Is(err, prayertime.ErrPolar) {
			return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
		}
		if err != nil {
			return nil, err
		}
		days = append(days, aladhan.Day{
			Date:  time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, m.Loc),
			Imsak: t.Imsak, Fajr: t.Fajr, Sunrise: t.Sunrise, Dhuhr: t.Dhuhr,
			Asr: t.Asr, Maghrib: t.Maghrib, Isha: t.Isha,
		})
	}
	return days, nil
}
