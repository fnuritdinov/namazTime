package daily

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	ayahs     []AyahRef
	reminders []Reminder
	overrides []Override
}

func (f fakeStore) Ayahs(context.Context) ([]AyahRef, error)      { return f.ayahs, nil }
func (f fakeStore) Reminders(context.Context) ([]Reminder, error) { return f.reminders, nil }
func (f fakeStore) Overrides(_ context.Context, from, to time.Time) ([]Override, error) {
	return f.overrides, nil
}

func day(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

var store = fakeStore{
	ayahs:     []AyahRef{{2, 255}, {2, 286}, {94, 5}},
	reminders: []Reminder{{1, map[string]string{"ru": "один"}}, {2, map[string]string{"ru": "два"}}},
}

func TestRotation(t *testing.T) {
	days, err := NewService(store).Days(context.Background(), day("2026-10-03"), day("2026-10-09"))
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 7 {
		t.Fatalf("got %d days, want 7", len(days))
	}
	// Соседние дни — соседние элементы круга
	for i := 1; i < len(days); i++ {
		if *days[i].Ayah == *days[i-1].Ayah {
			t.Errorf("%s and previous day have the same ayah", days[i].Date.Format(time.DateOnly))
		}
	}
	// Через 3 дня (длина списка аятов) аят повторяется
	if *days[3].Ayah != *days[0].Ayah {
		t.Errorf("ayah must repeat every 3 days: %v vs %v", days[3].Ayah, days[0].Ayah)
	}

	// Один и тот же день — один и тот же контент при любом запросе (для кеша и офлайна)
	one, _ := NewService(store).Days(context.Background(), day("2026-10-05"), day("2026-10-05"))
	if *one[0].Ayah != *days[2].Ayah || one[0].Reminder["ru"] != days[2].Reminder["ru"] {
		t.Error("the same date must give the same content in any range")
	}
}

func TestOverride(t *testing.T) {
	two := 2
	s := store
	s.overrides = []Override{
		{Date: day("2026-10-10"), Ayah: &AyahRef{97, 1}},                  // только аят
		{Date: day("2026-10-11"), ReminderID: &two},                       // только напоминание
		{Date: day("2026-10-12"), Ayah: &AyahRef{1, 1}, ReminderID: &two}, // оба
	}
	days, err := NewService(s).Days(context.Background(), day("2026-10-10"), day("2026-10-12"))
	if err != nil {
		t.Fatal(err)
	}
	if *days[0].Ayah != (AyahRef{97, 1}) {
		t.Errorf("10 Oct ayah = %v, want 97:1", days[0].Ayah)
	}
	if days[1].Reminder["ru"] != "два" {
		t.Errorf("11 Oct reminder = %v", days[1].Reminder)
	}
	if *days[2].Ayah != (AyahRef{1, 1}) || days[2].Reminder["ru"] != "два" {
		t.Errorf("12 Oct = %v %v", days[2].Ayah, days[2].Reminder)
	}
}

func TestEmptyAndErrors(t *testing.T) {
	days, err := NewService(fakeStore{}).Days(context.Background(), day("2026-10-03"), day("2026-10-03"))
	if err != nil || days[0].Ayah != nil || days[0].Reminder != nil {
		t.Errorf("empty store: %+v, %v", days, err)
	}

	var ie *InvalidError
	if _, err := NewService(store).Days(context.Background(), day("2026-10-09"), day("2026-10-03")); !errors.As(err, &ie) {
		t.Errorf("to < from: %v", err)
	}
	if _, err := NewService(store).Days(context.Background(), day("2026-10-01"), day("2026-11-15")); !errors.As(err, &ie) {
		t.Errorf("46 days: %v", err)
	}
}
