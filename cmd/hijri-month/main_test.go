package main

import "testing"

func TestParseInput(t *testing.T) {
	if _, err := parseInput("TJ", 1448, 9, "2027-02-08", true, "shuroiulamo-tj"); err != nil {
		t.Fatalf("valid input: %v", err)
	}
	bad := []struct {
		country string
		year    int
		month   int
		start   string
	}{
		{"tj", 1448, 9, "2027-02-08"},
		{"TJ", 2027, 9, "2027-02-08"},
		{"TJ", 1448, 13, "2027-02-08"},
		{"TJ", 1448, 9, "08.02.2027"},
	}
	for _, b := range bad {
		if _, err := parseInput(b.country, b.year, b.month, b.start, false, ""); err == nil {
			t.Errorf("%+v: expected error", b)
		}
	}
}

func TestShiftDays(t *testing.T) {
	cases := []struct {
		start   string
		month   int
		maxDiff int
	}{
		{"2027-02-08", 9, 1}, // Рамадан 1448 из ТЗ — совпадает с расчётом ±1
		{"2026-10-12", 5, 1}, // Джумада аль-уля 1448 по Шуро
		{"2026-09-13", 4, 1}, // Раби ас-сани 1448 по Шуро
	}
	for _, c := range cases {
		in, _ := parseInput("TJ", 1448, c.month, c.start, false, "")
		if d := shiftDays(in); abs(d) > c.maxDiff {
			t.Errorf("%s month %d: shift %d", c.start, c.month, d)
		}
	}
	// Опечатка в месяце: 2027-03-08 вместо 2027-02-08
	in, _ := parseInput("TJ", 1448, 9, "2027-03-08", false, "")
	if d := shiftDays(in); abs(d) <= maxShiftDays {
		t.Errorf("typo not detected, shift %d", d)
	}
}
