package prayertime

import (
	"testing"
	"time"
)

// Эталон из ТЗ (§6): Душанбе 38.5598, 68.787, MWL, 30.09.2026. Точность ±1 мин.
func TestCalculate_DushanbeReference(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Dushanbe")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := MethodByID("muslimWorldLeague")
	date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)

	tests := []struct {
		madhab string
		want   map[string]string
	}{
		{Hanafi, map[string]string{
			"imsak": "04:41", "fajr": "04:51", "sunrise": "06:20", "dhuhr": "12:17",
			"asr": "16:26", "maghrib": "18:11", "isha": "19:33",
		}},
		{Shafii, map[string]string{"asr": "15:36"}},
	}

	for _, tt := range tests {
		got, err := Calculate(date, 38.5598, 68.787, loc, m, tt.madhab)
		if err != nil {
			t.Fatal(err)
		}
		actual := map[string]time.Time{
			"imsak": got.Imsak, "fajr": got.Fajr, "sunrise": got.Sunrise, "dhuhr": got.Dhuhr,
			"asr": got.Asr, "maghrib": got.Maghrib, "isha": got.Isha,
		}
		for name, wantStr := range tt.want {
			want, _ := time.ParseInLocation("2006-01-02 15:04", "2026-09-30 "+wantStr, loc)
			diff := actual[name].Sub(want)
			if diff < -time.Minute || diff > time.Minute {
				t.Errorf("%s %s = %s, эталон %s (разница %v)",
					tt.madhab, name, actual[name].Format("15:04"), wantStr, diff)
			}
		}
	}
}

// Время должно идти по порядку и быть в часовом поясе города.
func TestCalculate_OrderAndZone(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Dushanbe")
	m, _ := MethodByID("muslimWorldLeague")
	got, err := Calculate(time.Date(2026, 9, 30, 0, 0, 0, 0, loc), 38.5598, 68.787, loc, m, Hanafi)
	if err != nil {
		t.Fatal(err)
	}
	seq := []time.Time{got.Imsak, got.Fajr, got.Sunrise, got.Dhuhr, got.Asr, got.Maghrib, got.Isha}
	for i := 1; i < len(seq); i++ {
		if !seq[i].After(seq[i-1]) {
			t.Errorf("нарушен порядок: %v не позже %v", seq[i], seq[i-1])
		}
	}
	if _, off := got.Fajr.Zone(); off != 5*3600 {
		t.Errorf("смещение %d, ожидали +05:00", off)
	}
}

// Умм аль-Кура: Иша = закат + 90 минут (Магриб = закат + 1 мин → разница 89 мин).
func TestCalculate_UmmAlQuraIsha(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Riyadh")
	m, _ := MethodByID("ummAlQura")
	got, err := Calculate(time.Date(2026, 9, 30, 0, 0, 0, 0, loc), 21.4225, 39.8262, loc, m, Shafii)
	if err != nil {
		t.Fatal(err)
	}
	if d := got.Isha.Sub(got.Maghrib); d < 88*time.Minute || d > 90*time.Minute {
		t.Errorf("Иша − Магриб = %v, ожидали ~89 мин", d)
	}
}

// Москва летом: солнце не опускается на 18°, без правила высоких широт было бы NaN.
func TestCalculate_HighLatitude(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	m, _ := MethodByID("muslimWorldLeague")
	got, err := Calculate(time.Date(2026, 6, 21, 0, 0, 0, 0, loc), 55.7558, 37.6173, loc, m, Hanafi)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fajr.Year() != 2026 || got.Isha.Year() != 2026 {
		t.Fatalf("некорректное время: fajr=%v isha=%v", got.Fajr, got.Isha)
	}
	if !got.Fajr.Before(got.Sunrise) || !got.Isha.After(got.Maghrib) {
		t.Errorf("fajr=%s sunrise=%s maghrib=%s isha=%s",
			got.Fajr.Format("15:04"), got.Sunrise.Format("15:04"), got.Maghrib.Format("15:04"), got.Isha.Format("15:04"))
	}
}
