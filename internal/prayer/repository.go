package prayer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound — записи в БД нет.
var ErrNotFound = errors.New("prayer times not found")

// Key — по чему ищем время намаза.
type Key struct {
	Lat, Lon float64
	Date     time.Time
	Method   int
	School   int
}

// Normalize округляет координаты до 0.01 (~1 км), чтобы соседние точки давали один ключ.
func (k Key) Normalize() Key {
	k.Lat = math.Round(k.Lat*100) / 100
	k.Lon = math.Round(k.Lon*100) / 100
	return k
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, k Key) (DayTimings, error) {
	const q = `
		SELECT date, source, COALESCE(hijri_date, ''),
		       to_char(fajr,    'HH24:MI'),
		       to_char(sunrise, 'HH24:MI'),
		       to_char(dhuhr,   'HH24:MI'),
		       to_char(asr,     'HH24:MI'),
		       to_char(maghrib, 'HH24:MI'),
		       to_char(isha,    'HH24:MI')
		FROM prayer_times
		WHERE lat = $1 AND lon = $2 AND date = $3 AND method = $4 AND school = $5`

	var d DayTimings
	err := r.db.QueryRow(ctx, q, k.Lat, k.Lon, k.Date, k.Method, k.School).Scan(
		&d.Date, &d.Source, &d.HijriDate,
		&d.Timings.Fajr, &d.Timings.Sunrise, &d.Timings.Dhuhr,
		&d.Timings.Asr, &d.Timings.Maghrib, &d.Timings.Isha,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DayTimings{}, ErrNotFound
	}
	if err != nil {
		return DayTimings{}, fmt.Errorf("select prayer_times: %w", err)
	}
	return d, nil
}

func (r *Repository) Save(ctx context.Context, k Key, d DayTimings) error {
	const q = `
		INSERT INTO prayer_times
		    (lat, lon, date, method, school, source, hijri_date,
		     fajr, sunrise, dhuhr, asr, maghrib, isha)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''),
		        $8::text::time, $9::text::time, $10::text::time,
		        $11::text::time, $12::text::time, $13::text::time)
		ON CONFLICT ON CONSTRAINT prayer_times_unique DO NOTHING`

	t := d.Timings
	_, err := r.db.Exec(ctx, q,
		k.Lat, k.Lon, k.Date, k.Method, k.School, d.Source, d.HijriDate,
		t.Fajr, t.Sunrise, t.Dhuhr, t.Asr, t.Maghrib, t.Isha,
	)
	if err != nil {
		return fmt.Errorf("insert prayer_times: %w", err)
	}
	return nil
}
