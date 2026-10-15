// shuro-sync — разовая загрузка месяца с сайта Шурои уламо (то же, что делает воркер).
//
//	go run ./cmd/shuro-sync -month 2026-11          // только скачать и показать
//	go run ./cmd/shuro-sync -month 2026-11 -save    // скачать и записать в базу (нужен DATABASE_URL)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nTime/internal/shuro"
)

func main() {
	month := flag.String("month", "", "месяц YYYY-MM, например 2026-11")
	save := flag.Bool("save", false, "записать в базу")
	city := flag.String("city", "dushanbe", "id города")
	source := flag.String("source", "shuroiulamo-tj", "id источника")
	flag.Parse()

	m, err := time.Parse("2006-01", *month)
	if err != nil {
		log.Fatalf("-month: expected YYYY-MM, got %q", *month)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rows, err := shuro.NewClient().FetchMonth(ctx, m.Year(), m.Month())
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}
	for _, r := range rows {
		fmt.Printf("%s %02d.%02d.%d  %s-%s  %s-%s  %s-%s  %s-%s  %s-%s  %s-%s\n",
			r.Date.Format("02.01.2006"), r.HijriDay, r.HijriMonth, r.HijriYear,
			r.FajrStart, r.FajrEnd, r.DhuhrStart, r.DhuhrEnd, r.AsrStart, r.AsrEnd,
			r.MakruhStart, r.MakruhEnd, r.MaghribStart, r.MaghribEnd, r.IshaStart, r.IshaEnd)
	}
	fmt.Printf("fetched %d days\n", len(rows))
	if !*save {
		return
	}

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()
	if err := (&shuro.Store{DB: db}).SaveMonth(ctx, *city, *source, rows); err != nil {
		log.Fatalf("save: %v", err)
	}
	fmt.Println("saved")
}
