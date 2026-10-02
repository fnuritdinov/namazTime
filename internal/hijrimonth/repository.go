package hijrimonth

import (
	"context"
	"fmt"
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
