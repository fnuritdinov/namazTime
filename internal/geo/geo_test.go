package geo

import "testing"

func TestCountry(t *testing.T) {
	l, err := New()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		lat, lon float64
		want     string
	}{
		{"Ташкент", 41.31, 69.24, "UZ"},
		{"Душанбе", 38.56, 68.77, "TJ"},
		{"Худжанд", 40.28, 69.62, "TJ"},
		{"Москва", 55.75, 37.62, "RU"},
		{"Мекка", 21.42, 39.83, "SA"},
		{"Индийский океан", -20.0, 80.0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := l.Country(tt.lat, tt.lon); got != tt.want {
				t.Errorf("Country(%v, %v) = %q, want %q", tt.lat, tt.lon, got, tt.want)
			}
		})
	}
}
