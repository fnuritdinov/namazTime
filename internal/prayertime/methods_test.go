package prayertime

import "testing"

func TestMethodsUnique(t *testing.T) {
	ids := map[string]bool{}
	aladhan := map[int]string{}
	for _, m := range Methods {
		if ids[m.ID] {
			t.Errorf("duplicate ID %q", m.ID)
		}
		ids[m.ID] = true
		if prev, ok := aladhan[m.AladhanID]; ok {
			t.Errorf("AladhanID %d used by %s and %s", m.AladhanID, prev, m.ID)
		}
		aladhan[m.AladhanID] = m.ID
		if m.Title["en"] == "" || m.Title["ru"] == "" || m.Title["tg"] == "" {
			t.Errorf("%s: title en/ru/tg required", m.ID)
		}
	}
	if len(Methods) != 23 {
		t.Errorf("want 23 methods, got %d", len(Methods))
	}
}
