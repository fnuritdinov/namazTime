// import-timetable загружает официальное расписание намазов Шуро из CSV в Postgres.
//
// Пример:
//
//	go run ./cmd/import-timetable -city dushanbe -file data/dushanbe-2026-10.csv
//
// Формат CSV — как строка таблицы на shuroiulamo.tj (первая строка — заголовок):
//
//	date,hijri,fajr_start,fajr_end,dhuhr_start,dhuhr_end,asr_start,asr_end,makruh_start,makruh_end,maghrib_start,maghrib_end,isha_start,isha_end
//	01.10.2026,19.04.1448,04:51,06:17,12:40,16:24,16:25,17:58,17:59,18:18,18:19,18:59,19:49,04:50
//
// isha_end — время СЛЕДУЮЩЕГО утра. Восход = fajr_end.
// Повторный импорт того же файла безопасен: строки обновляются, а не дублируются.
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var header = []string{
	"date", "hijri",
	"fajr_start", "fajr_end",
	"dhuhr_start", "dhuhr_end",
	"asr_start", "asr_end",
	"makruh_start", "makruh_end",
	"maghrib_start", "maghrib_end",
	"isha_start", "isha_end",
}

// timeCols — колонки со временем (всё, кроме date и hijri), в порядке header.
var timeCols = header[2:]

type day struct {
	date                            time.Time
	t                               map[string]string // колонка → "HH:MM"
	hijriDay, hijriMonth, hijriYear int
}

func main() {
	cityID := flag.String("city", "", "id города, например dushanbe")
	sourceID := flag.String("source", "shuroiulamo-tj", "id источника из official_sources")
	file := flag.String("file", "", "путь к CSV-файлу")
	flag.Parse()

	if *cityID == "" || *file == "" {
		flag.Usage()
		os.Exit(2)
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	days, err := readCSV(*file)
	if err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}

	n, err := importDays(context.Background(), dbURL, *cityID, *sourceID, days)
	if err != nil {
		log.Fatalf("import: %v", err)
	}
	fmt.Printf("imported %d days for %s (source %s)\n", n, *cityID, *sourceID)
}

// readCSV читает и ПРОВЕРЯЕТ файл целиком до того, как что-то писать в базу.
func readCSV(path string) ([]day, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = len(header)

	head, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	head[0] = strings.TrimPrefix(head[0], "\uFEFF") // Excel добавляет BOM в начало файла
	for i, h := range header {
		if strings.TrimSpace(head[i]) != h {
			return nil, fmt.Errorf("header: column %d must be %q, got %q", i+1, h, head[i])
		}
	}

	var days []day
	for line := 2; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		d, err := parseRow(rec)
		if err != nil {
			return nil, fmt.Errorf("line %d (%s): %w", line, rec[0], err)
		}
		if len(days) > 0 && d.date.Year() != days[0].date.Year() {
			return nil, fmt.Errorf("line %d: all dates must be in the same year", line)
		}
		days = append(days, d)
	}
	if len(days) == 0 {
		return nil, errors.New("no data rows")
	}
	return days, nil
}

func parseRow(rec []string) (day, error) {
	for i := range rec {
		rec[i] = strings.TrimSpace(rec[i])
	}

	date, err := time.Parse("02.01.2006", rec[0])
	if err != nil {
		return day{}, fmt.Errorf("date %q: expected DD.MM.YYYY", rec[0])
	}

	d := day{date: date, t: map[string]string{}}

	// Хиджра "19.04.1448"
	hp := strings.Split(rec[1], ".")
	if len(hp) != 3 {
		return day{}, fmt.Errorf("hijri %q: expected DD.MM.YYYY", rec[1])
	}
	nums := []*int{&d.hijriDay, &d.hijriMonth, &d.hijriYear}
	for i, s := range hp {
		v, err := strconv.Atoi(s)
		if err != nil {
			return day{}, fmt.Errorf("hijri %q: expected DD.MM.YYYY", rec[1])
		}
		*nums[i] = v
	}
	if d.hijriDay < 1 || d.hijriDay > 30 || d.hijriMonth < 1 || d.hijriMonth > 12 {
		return day{}, fmt.Errorf("hijri %q: day must be 1..30, month 1..12", rec[1])
	}

	// Время: формат HH:MM
	for i, name := range timeCols {
		v := rec[2+i]
		if _, err := time.Parse("15:04", v); err != nil {
			return day{}, fmt.Errorf("%s %q: expected HH:MM", name, v)
		}
		d.t[name] = v
	}

	// Порядок за день: fajr_start < fajr_end < dhuhr_start < ... < isha_start.
	// "HH:MM" с ведущими нулями можно сравнивать как строки.
	// isha_end не проверяем по порядку — это время следующего утра.
	order := timeCols[:len(timeCols)-1]
	for i := 1; i < len(order); i++ {
		prev, cur := order[i-1], order[i]
		if d.t[cur] <= d.t[prev] {
			return day{}, fmt.Errorf("%s (%s) must be later than %s (%s)", cur, d.t[cur], prev, d.t[prev])
		}
	}
	if d.t["isha_end"] >= "12:00" {
		return day{}, fmt.Errorf("isha_end (%s) must be the next morning (before 12:00)", d.t["isha_end"])
	}
	return d, nil
}

// importDays пишет всё в ОДНОЙ транзакции: либо загрузится весь файл, либо ничего.
func importDays(ctx context.Context, dbURL, cityID, sourceID string, days []day) (int, error) {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return 0, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) // после Commit ничего не делает

	var timetableID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO official_timetables (city_id, source_id, year)
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT official_timetables_unique DO UPDATE SET updated_at = now()
		RETURNING id`,
		cityID, sourceID, days[0].date.Year(),
	).Scan(&timetableID)
	if err != nil {
		return 0, fmt.Errorf("upsert timetable: %w", err)
	}

	// Начала намазов — в старые колонки (fajr, dhuhr, ...), восход = конец Бомдода.
	const upsertDay = `
		INSERT INTO official_timetable_days
		    (timetable_id, date,
		     fajr, fajr_end, sunrise, dhuhr, dhuhr_end, asr, asr_end,
		     makruh_start, makruh_end, maghrib, maghrib_end, isha, isha_end,
		     hijri_day, hijri_month, hijri_year)
		VALUES ($1, $2,
		        $3::text::time, $4::text::time, $4::text::time, $5::text::time, $6::text::time,
		        $7::text::time, $8::text::time, $9::text::time, $10::text::time,
		        $11::text::time, $12::text::time, $13::text::time, $14::text::time,
		        $15, $16, $17)
		ON CONFLICT (timetable_id, date) DO UPDATE SET
		    fajr = EXCLUDED.fajr, fajr_end = EXCLUDED.fajr_end, sunrise = EXCLUDED.sunrise,
		    dhuhr = EXCLUDED.dhuhr, dhuhr_end = EXCLUDED.dhuhr_end,
		    asr = EXCLUDED.asr, asr_end = EXCLUDED.asr_end,
		    makruh_start = EXCLUDED.makruh_start, makruh_end = EXCLUDED.makruh_end,
		    maghrib = EXCLUDED.maghrib, maghrib_end = EXCLUDED.maghrib_end,
		    isha = EXCLUDED.isha, isha_end = EXCLUDED.isha_end,
		    hijri_day = EXCLUDED.hijri_day, hijri_month = EXCLUDED.hijri_month, hijri_year = EXCLUDED.hijri_year`

	for _, d := range days {
		t := d.t
		if _, err := tx.Exec(ctx, upsertDay,
			timetableID, d.date,
			t["fajr_start"], t["fajr_end"], t["dhuhr_start"], t["dhuhr_end"],
			t["asr_start"], t["asr_end"], t["makruh_start"], t["makruh_end"],
			t["maghrib_start"], t["maghrib_end"], t["isha_start"], t["isha_end"],
			d.hijriDay, d.hijriMonth, d.hijriYear,
		); err != nil {
			return 0, fmt.Errorf("day %s: %w", d.date.Format("2006-01-02"), err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE cities SET has_official_timetable = true WHERE id = $1`, cityID); err != nil {
		return 0, fmt.Errorf("update city: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return len(days), nil
}
