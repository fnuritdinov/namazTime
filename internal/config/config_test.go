package config

import (
	"reflect"
	"testing"
)

func TestParseVersions(t *testing.T) {
	got, err := parseVersions(" cities=7, quran=3 ,hadith=2,")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"cities": 7, "quran": 3, "hadith": 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, bad := range []string{"cities", "cities=x", "=3", "cities=-1"} {
		if _, err := parseVersions(bad); err == nil {
			t.Errorf("%q должно быть ошибкой", bad)
		}
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinAppVersion != "1.0.0" || cfg.FeatureSync || cfg.ContentVersions["cities"] != 1 {
		t.Errorf("%+v", cfg)
	}

	t.Setenv("FEATURE_SYNC", "yes")
	if _, err := Load(); err == nil {
		t.Error("FEATURE_SYNC=yes должно быть ошибкой (только true/false)")
	}
}
