package official

import "testing"

func goodRow() Row {
	return Row{
		HijriDay: 6, HijriMonth: 6, HijriYear: 1448,
		FajrStart: "05:38", FajrEnd: "07:05",
		DhuhrStart: "12:40", DhuhrEnd: "15:30",
		AsrStart: "15:31", AsrEnd: "17:01",
		MakruhStart: "17:02", MakruhEnd: "17:20",
		MaghribStart: "17:21", MaghribEnd: "18:05",
		IshaStart: "18:55", IshaEnd: "05:37",
	}
}

func TestValidate_OK(t *testing.T) {
	if err := goodRow().Validate(); err != nil {
		t.Fatal(err)
	}
}

// Шуро, 17.11.2026: конец Асра и начало макрӯҳ — одна минута. Это не должно отклонять месяц.
func TestValidate_BoundaryMayTouch(t *testing.T) {
	r := goodRow()
	r.MakruhStart = r.AsrEnd
	if err := r.Validate(); err != nil {
		t.Fatalf("touching boundary must be allowed: %v", err)
	}
}

func TestValidate_Broken(t *testing.T) {
	cases := map[string]func(*Row){
		"end equals own start":   func(r *Row) { r.AsrEnd = r.AsrStart },
		"next starts before end": func(r *Row) { r.MakruhStart = "17:00" },
		"bad format":             func(r *Row) { r.DhuhrStart = "12-40" },
		"isha end not morning":   func(r *Row) { r.IshaEnd = "23:00" },
		"bad hijri":              func(r *Row) { r.HijriMonth = 13 },
	}
	for name, change := range cases {
		r := goodRow()
		change(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
