package prayertime

import "testing"

func TestMethods(t *testing.T) {
	// Идентификаторы из ТЗ и iOS — ни одного лишнего, ни одного пропущенного
	want := []string{"muslimWorldLeague", "northAmerica", "egypt", "ummAlQura", "karachi", "turkey"}

	if len(Methods) != len(want) {
		t.Fatalf("методов %d, ожидали %d", len(Methods), len(want))
	}
	for _, id := range want {
		m, ok := MethodByID(id)
		if !ok {
			t.Errorf("метод %q не найден", id)
			continue
		}
		if m.Title["en"] == "" {
			t.Errorf("у метода %q нет английского названия (фолбэк по ТЗ)", id)
		}
		if m.FajrAngle <= 0 {
			t.Errorf("у метода %q не задан угол Фаджра", id)
		}
	}

	if _, ok := MethodByID("unknown"); ok {
		t.Error("несуществующий метод не должен находиться")
	}
}
