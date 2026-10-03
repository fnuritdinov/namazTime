package hijrimonth

import (
	"context"
	"fmt"
	"nTime/internal/hijri"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Months(ctx context.Context, country string, hijriYear int) ([]Month, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.month, m.start_date, m.status, m.confirmed_at, COALESCE(s.name, '{}'::jsonb)
		FROM hijri_months m
		LEFT JOIN official_sources s ON s.id = m.source_id
		WHERE m.country = $1 AND m.hijri_year = $2
		ORDER BY m.month`,
		country, hijriYear)
	if err != nil {
		return nil, fmt.Errorf("select hijri months: %w", err)
	}
	defer rows.Close()

	months := []Month{}
	for rows.Next() {
		var m Month
		var confirmedAt *time.Time
		if err := rows.Scan(&m.Month, &m.Start, &m.Status, &confirmedAt, &m.SourceName); err != nil {
			return nil, fmt.Errorf("scan hijri month: %w", err)
		}
		m.ConfirmedAt = confirmedAt
		months = append(months, m)
	}
	return months, rows.Err()
}

func (r *Repository) Duas(ctx context.Context, category string) ([]Dua, error) {
	// LEFT JOIN: дуа без переводов тоже попадёт в результат
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.arabic, COALESCE(d.transliteration, ''), COALESCE(d.source, ''),
		       COALESCE(t.lang, ''), COALESCE(t.text, '')
		FROM duas d
		LEFT JOIN dua_translations t ON t.dua_id = d.id
		WHERE d.category = $1
		ORDER BY d.sort_order, d.id`,
		category)
	if err != nil {
		return nil, fmt.Errorf("select duas: %w", err)
	}
	defer rows.Close()

	duas := []Dua{}
	index := map[string]int{} // id → позиция в duas
	for rows.Next() {
		var d Dua
		var lang, text string
		if err := rows.Scan(&d.ID, &d.Arabic, &d.Transliteration, &d.Source, &lang, &text); err != nil {
			return nil, fmt.Errorf("scan dua: %w", err)
		}
		i, ok := index[d.ID]
		if !ok {
			d.Translations = map[string]string{}
			duas = append(duas, d)
			i = len(duas) - 1
			index[d.ID] = i
		}
		if lang != "" {
			duas[i].Translations[lang] = text
		}
	}
	return duas, rows.Err()
}

// Starts — начала месяцев страны с start_date в [from, to] (для календаря в /prayer-times).
// Берём и ожидаемые, и подтверждённые: дата Шуро точнее табличного расчёта.
func (r *Repository) Starts(ctx context.Context, country string, from, to time.Time) ([]hijri.MonthStart, error) {
	rows, err := r.db.Query(ctx, `
		SELECT hijri_year, month, start_date
		FROM hijri_months
		WHERE country = $1 AND start_date BETWEEN $2 AND $3
		ORDER BY start_date`,
		country, from, to)
	if err != nil {
		return nil, fmt.Errorf("select hijri month starts: %w", err)
	}
	defer rows.Close()

	var starts []hijri.MonthStart
	for rows.Next() {
		var m hijri.MonthStart
		if err := rows.Scan(&m.Year, &m.Month, &m.Start); err != nil {
			return nil, fmt.Errorf("scan hijri month start: %w", err)
		}
		starts = append(starts, m)
	}
	return starts, rows.Err()
}
