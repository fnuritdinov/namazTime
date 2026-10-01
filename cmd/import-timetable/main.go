// import-timetable загружает официальное расписание намазов из CSV в Postgres.
//
// Пример:
//
//	go run ./cmd/import-timetable -city dushanbe -file data/dushanbe-2026-10.csv
//
// Формат CSV (первая строка — заголовок):
//
//	date,fajr,sunrise,dhuhr,asr,maghrib,isha,hijri_day,hijri_month,hijri_year
//	2026-10-01,04:51,06:17,12:40,16:25,18:19,19:49,19,4,1448
//
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

var header = []string{"date", "fajr", "sunrise", "dhuhr", "asr", "maghrib", "isha", "hijri_day", "hijri_month", "hijri_year"}

type day struct {
	date                                     time.Time
	fajr, sunrise, dhuhr, asr, maghrib, isha string
	hijriDay, hijriMonth, hijriYear          *int
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
			return nil, fmt.Errorf("line %d: %w", line, err)
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

	date, err := time.Parse("2006-01-02", rec[0])
	if err != nil {
		return day{}, fmt.Errorf("date %q: expected YYYY-MM-DD", rec[0])
	}

	times := rec[1:7]
	names := header[1:7]
	for i, t := range times {
		if _, err := time.Parse("15:04", t); err != nil {
			return day{}, fmt.Errorf("%s %q: expected HH:MM", names[i], t)
		}
		// Намазы должны идти по порядку: fajr < sunrise < dhuhr < asr < maghrib < isha.
		// "HH:MM" с ведущими нулями можно сравнивать как строки.
		if i > 0 && t <= times[i-1] {
			return day{}, fmt.Errorf("%s (%s) must be later than %s (%s)", names[i], t, names[i-1], times[i-1])
		}
	}

	d := day{
		date: date,
		fajr: times[0], sunrise: times[1], dhuhr: times[2],
		asr: times[3], maghrib: times[4], isha: times[5],
	}

	// Хиджра необязательна: пустые ячейки → NULL
	hijri := []**int{&d.hijriDay, &d.hijriMonth, &d.hijriYear}
	for i, s := range rec[7:10] {
		if s == "" {
			continue
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return day{}, fmt.Errorf("%s %q: expected a number", header[7+i], s)
		}
		*hijri[i] = &v
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

	const upsertDay = `
		INSERT INTO official_timetable_days
		    (timetable_id, date, fajr, sunrise, dhuhr, asr, maghrib, isha, hijri_day, hijri_month, hijri_year)
		VALUES ($1, $2, $3::text::time, $4::text::time, $5::text::time,
		        $6::text::time, $7::text::time, $8::text::time, $9, $10, $11)
		ON CONFLICT (timetable_id, date) DO UPDATE SET
		    fajr = EXCLUDED.fajr, sunrise = EXCLUDED.sunrise, dhuhr = EXCLUDED.dhuhr,
		    asr = EXCLUDED.asr, maghrib = EXCLUDED.maghrib, isha = EXCLUDED.isha,
		    hijri_day = EXCLUDED.hijri_day, hijri_month = EXCLUDED.hijri_month, hijri_year = EXCLUDED.hijri_year`

	for _, d := range days {
		if _, err := tx.Exec(ctx, upsertDay,
			timetableID, d.date, d.fajr, d.sunrise, d.dhuhr, d.asr, d.maghrib, d.isha,
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
