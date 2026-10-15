package shuro

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"nTime/internal/official"
)

// Store сохраняет месяц в official_timetable_days — тем же SQL, что и cmd/import-timetable.
type Store struct {
	DB *pgxpool.Pool
}

// SaveMonth пишет все дни в ОДНОЙ транзакции: либо весь месяц, либо ничего.
func (s *Store) SaveMonth(ctx context.Context, cityID, sourceID string, rows []official.Row) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // после Commit ничего не делает

	var timetableID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO official_timetables (city_id, source_id, year)
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT official_timetables_unique DO UPDATE SET updated_at = now()
		RETURNING id`,
		cityID, sourceID, rows[0].Date.Year(),
	).Scan(&timetableID)
	if err != nil {
		return fmt.Errorf("upsert timetable: %w", err)
	}

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

	for _, r := range rows {
		if _, err := tx.Exec(ctx, upsertDay,
			timetableID, r.Date,
			r.FajrStart, r.FajrEnd, r.DhuhrStart, r.DhuhrEnd,
			r.AsrStart, r.AsrEnd, r.MakruhStart, r.MakruhEnd,
			r.MaghribStart, r.MaghribEnd, r.IshaStart, r.IshaEnd,
			r.HijriDay, r.HijriMonth, r.HijriYear,
		); err != nil {
			return fmt.Errorf("day %s: %w", r.Date.Format("2006-01-02"), err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE cities SET has_official_timetable = true WHERE id = $1`, cityID); err != nil {
		return fmt.Errorf("update city: %w", err)
	}
	return tx.Commit(ctx)
}
