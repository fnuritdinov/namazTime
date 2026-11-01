package takvim

import (
	"strings"
	"testing"
)

// Кусок настоящей страницы takvim.tj (01.11.2026). Блок case 5 (другой город) не должен попасть в результат.
const samplePage = `
switch (shift_for_namaz) {
/* Душанбе, Вахдат, Турсунзаде, Хисор*/
			case 0:
				 var date_imruz         = "01-11-2026"
				 var bomdod_from_imruz  = "05:22";
				 var bomdod_to_imruz    = "06:48"
				 var peshin_from_imruz  = "12:40"
				 var peshin_to_imruz    = "15:41"
				 var asr_from_imruz     = "15:42"
				 var asr_to_imruz       = "17:14"
				 var forbidden_time_from_imruz = "17:15"
				 var forbidden_time_to_imruz = "17:34"
				 var shom_from_imruz    = "17:35"
				 var shom_to_imruz      = "18:15"
				 var khuftan_from_imruz = "19:05"
				 var khuftan_to_imruz   = "05:21"
				 var date_pagoh         = "02-11-2026"
				 var bomdod_from_pagoh  = "05:23"
				 var bomdod_to_pagoh    = "06:49"
				 var peshin_from_pagoh  = "12:40"
				 var peshin_to_pagoh    = "15:40"
				 var asr_from_pagoh     = "15:41"
				 var asr_to_pagoh       = "17:13"
				 var forbidden_time_from_pagoh = "17:14"
				 var forbidden_time_to_pagoh = "17:33"
				 var shom_from_pagoh    = "17:34"
				 var shom_to_pagoh      = "18:14"
				 var khuftan_from_pagoh = "19:04"
				 var khuftan_to_pagoh   = "05:22"
				 var date_pagohi_digar         = "03-11-2026"
				 var bomdod_from_pagohi_digar  = "05:24"
				 var bomdod_to_pagohi_digar    = "06:50"
				 var peshin_from_pagohi_digar  = "12:40"
				 var peshin_to_pagohi_digar    = "15:39"
				 var asr_from_pagohi_digar     = "15:40"
				 var asr_to_pagohi_digar       = "17:12"
				 var forbidden_time_from_pagohi_digar = "17:13"
				 var forbidden_time_to_pagohi_digar = "17:32"
				 var shom_from_pagohi_digar    = "17:33"
				 var shom_to_pagohi_digar      = "18:13"
				 var khuftan_from_pagohi_digar = "19:03"
				 var khuftan_to_pagohi_digar   = "05:23"
		 break;
			case 5:
				 var date_imruz         = "01-11-2026"
				 var bomdod_from_imruz  = "05:27";
		 break;
}`

func TestParse(t *testing.T) {
	rows, err := Parse(samplePage)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	r := rows[0]
	if r.Date.Format("2006-01-02") != "2026-11-01" || r.FajrStart != "05:22" || r.FajrEnd != "06:48" ||
		r.MakruhStart != "17:15" || r.MaghribStart != "17:35" || r.IshaStart != "19:05" || r.IshaEnd != "05:21" {
		t.Errorf("today parsed wrong: %+v", r)
	}
	if rows[2].Date.Format("2006-01-02") != "2026-11-03" || rows[2].AsrEnd != "17:12" {
		t.Errorf("after tomorrow parsed wrong: %+v", rows[2])
	}
}

func TestParse_Broken(t *testing.T) {
	if _, err := Parse("<html>сайт поменялся</html>"); err == nil {
		t.Error("want error for page without data")
	}
	broken := strings.Replace(samplePage, `"15:41"`, `"xx:41"`, 1)
	if _, err := Parse(broken); err == nil {
		t.Error("want error for bad time")
	}
}
