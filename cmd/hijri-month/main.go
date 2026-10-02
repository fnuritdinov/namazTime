// hijri-month сохраняет начало месяца хиджры для страны.
//
// Ожидаемая дата (по календарю Шуро, до наблюдения луны):
//
//	go run ./cmd/hijri-month -country TJ -year 1448 -month 9 -start 2027-02-08
//
// Подтверждённая дата (Шуро объявили после наблюдения луны):
//
//	go run ./cmd/hijri-month -country TJ -year 1448 -month 9 -start 2027-02-08 -confirm
//
// Повторный запуск для того же месяца обновляет запись, а не дублирует её.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"nTime/internal/hijri"
)

// input — проверенные параметры командной строки.
type input struct {
	country string
	year    int
	month   int
	start   time.Time
	confirm bool
	source  string
}

var countryRe = regexp.MustCompile(`^[A-Z]{2}$`)

// maxShiftDays — насколько дата может отличаться от табличного календаря.
// Реально отличие 1–2 дня; больше — почти наверняка опечатка.
const maxShiftDays = 3

func main() {
	country := flag.String("country", "", "код страны ISO 3166-1, например TJ")
	year := flag.Int("year", 0, "год хиджры, например 1448")
	month := flag.Int("month", 0, "месяц хиджры 1..12 (9 — Рамадан, 10 — Шавваль)")
	start := flag.String("start", "", "дата начала месяца, YYYY-MM-DD")
	confirm := flag.Bool("confirm", false, "дата подтверждена (луну увидели)")
	source := flag.String("source", "shuroiulamo-tj", "id источника из official_sources; пусто — без источника")
	force := flag.Bool("force", false, "сохранить, даже если дата сильно отличается от расчётной")
	flag.Parse()

	in, err := parseInput(*country, *year, *month, *start, *confirm, *source)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(2)
	}

	if shift := shiftDays(in); abs(shift) > maxShiftDays && !*force {
		log.Fatalf("%s отличается от расчётной даты на %d дн. — проверь опечатку или добавь -force", *start, shift)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := save(context.Background(), dbURL, in); err != nil {
		log.Fatalf("save: %v", err)
	}

	status := "expected"
	if in.confirm {
		status = "confirmed"
	}
	fmt.Printf("saved: %s %d/%d (%s) starts %s, %s\n",
		in.country, in.year, in.month, hijri.Date{Month: in.month}.MonthName()["en"],
		in.start.Format("2006-01-02"), status)
}

func parseInput(country string, year, month int, start string, confirm bool, source string) (input, error) {
	if !countryRe.MatchString(country) {
		return input{}, fmt.Errorf("-country must be 2 capital letters, e.g. TJ")
	}
	if year < 1400 || year > 1600 {
		return input{}, fmt.Errorf("-year must be between 1400 and 1600")
	}
	if month < 1 || month > 12 {
		return input{}, fmt.Errorf("-month must be between 1 and 12")
	}
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return input{}, fmt.Errorf("-start %q: expected YYYY-MM-DD", start)
	}
	return input{country: country, year: year, month: month, start: t, confirm: confirm, source: source}, nil
}

// shiftDays — на сколько дней дата отличается от табличного календаря.
func shiftDays(in input) int {
	tabular := hijri.ToGregorian(hijri.Date{Day: 1, Month: in.month, Year: in.year})
	return int(in.start.Sub(tabular).Hours() / 24)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func save(ctx context.Context, dbURL string, in input) error {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	// confirmed_at ставим только для подтверждённой даты (этого требует CHECK в таблице).
	// NULLIF('', '') → NULL: пустой -source значит «без источника».
	_, err = conn.Exec(ctx, `
		INSERT INTO hijri_months (country, hijri_year, month, start_date, status, confirmed_at, source_id)
		VALUES ($1, $2, $3, $4,
		        CASE WHEN $5 THEN 'confirmed' ELSE 'expected' END,
		        CASE WHEN $5 THEN now() END,
		        NULLIF($6, ''))
		ON CONFLICT (country, hijri_year, month) DO UPDATE SET
		    start_date   = EXCLUDED.start_date,
		    status       = EXCLUDED.status,
		    confirmed_at = EXCLUDED.confirmed_at,
		    source_id    = EXCLUDED.source_id`,
		in.country, in.year, in.month, in.start, in.confirm, in.source)
	return err
}
