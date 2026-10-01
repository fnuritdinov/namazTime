package official

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Get возвращает официальное расписание города на [from, to].
// Если у города задан official_base_city_id (например, Худжанд → Душанбе),
// берётся таблица базового города, а поправки — самого города.
func (r *Repository) Get(ctx context.Context, cityID string, from, to time.Time) (Timetable, error) {
	// 1. Чью таблицу брать
	var baseCityID string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(official_base_city_id, id) FROM cities WHERE id = $1`, cityID,
	).Scan(&baseCityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Timetable{}, ErrNotFound
	}
	if err != nil {
		return Timetable{}, fmt.Errorf("resolve base city: %w", err)
	}

	// 2. Дни расписания
	rows, err := r.db.Query(ctx, `
		SELECT d.date,
		       to_char(d.fajr, 'HH24:MI'), to_char(d.sunrise, 'HH24:MI'), to_char(d.dhuhr, 'HH24:MI'),
		       to_char(d.asr, 'HH24:MI'), to_char(d.maghrib, 'HH24:MI'), to_char(d.isha, 'HH24:MI'),
		       COALESCE(to_char(d.imsak, 'HH24:MI'), ''),
		       COALESCE(d.hijri_day, 0), COALESCE(d.hijri_month, 0), COALESCE(d.hijri_year, 0),
		       t.updated_at, s.name
		FROM official_timetable_days d
		JOIN official_timetables t ON t.id = d.timetable_id
		JOIN official_sources s    ON s.id = t.source_id
		WHERE t.city_id = $1 AND d.date BETWEEN $2 AND $3
		ORDER BY d.date`,
		baseCityID, from, to)
	if err != nil {
		return Timetable{}, fmt.Errorf("select official days: %w", err)
	}
	defer rows.Close()

	tt := Timetable{Days: map[string]Day{}}
	for rows.Next() {
		var (
			date      time.Time
			d         Day
			updatedAt time.Time
			name      map[string]string
		)
		if err := rows.Scan(&date,
			&d.Fajr, &d.Sunrise, &d.Dhuhr, &d.Asr, &d.Maghrib, &d.Isha, &d.Imsak,
			&d.HijriDay, &d.HijriMonth, &d.HijriYear,
			&updatedAt, &name,
		); err != nil {
			return Timetable{}, fmt.Errorf("scan official day: %w", err)
		}
		tt.Days[date.Format("2006-01-02")] = d
		tt.SourceName = name
		if updatedAt.After(tt.UpdatedAt) {
			tt.UpdatedAt = updatedAt
		}
	}
	if err := rows.Err(); err != nil {
		return Timetable{}, err
	}
	if len(tt.Days) == 0 {
		return Timetable{}, ErrNotFound
	}

	// 3. Поправки ± минут для запрошенного города
	adj, err := r.adjustments(ctx, cityID)
	if err != nil {
		return Timetable{}, err
	}
	tt.Adjustments = adj
	return tt, nil
}

func (r *Repository) adjustments(ctx context.Context, cityID string) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `SELECT prayer, minutes FROM time_adjustments WHERE city_id = $1`, cityID)
	if err != nil {
		return nil, fmt.Errorf("select adjustments: %w", err)
	}
	defer rows.Close()

	adj := map[string]int{}
	for rows.Next() {
		var prayer string
		var minutes int
		if err := rows.Scan(&prayer, &minutes); err != nil {
			return nil, fmt.Errorf("scan adjustment: %w", err)
		}
		adj[prayer] = minutes
	}
	return adj, rows.Err()
}
