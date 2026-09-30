package qibla

import (
	"math"
	"testing"
)

func TestDirection(t *testing.T) {
	tests := []struct {
		name     string
		lat, lon float64
		want     float64
	}{
		{"Ташкент", 41.31, 69.24, 240.3},
		{"Душанбе", 38.56, 68.77, 243.7},
		{"Москва", 55.75, 37.62, 176.4},
		{"Лондон", 51.5074, -0.1278, 119.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Direction(tt.lat, tt.lon)
			if math.Abs(got-tt.want) > 0.5 {
				t.Errorf("Direction(%v, %v) = %.1f, want ≈%.0f", tt.lat, tt.lon, got, tt.want)
			}
		})
	}
}
