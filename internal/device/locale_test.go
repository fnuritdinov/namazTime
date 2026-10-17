package device

import "testing"

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"ru": "ru", "ru_TJ": "ru_TJ", "ru-TJ": "ru_TJ", "tg_TJ": "tg_TJ",
		"en_US@rg=gbzzzz": "en_US", "ru_TJ@calendar=islamic": "ru_TJ",
		"zh-Hans_CN": "zh_CN", "sr-Latn": "sr", "es_419": "es_419",
		"RU_tj": "ru_TJ", "": "", "123": "", "русский": "",
	}
	for in, want := range cases {
		if got := normalizeLocale(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
