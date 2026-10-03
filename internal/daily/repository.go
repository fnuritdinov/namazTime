package daily

import (
	"context"
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

func (r *Repository) Ayahs(ctx context.Context) ([]AyahRef, error) {
	rows, err := r.db.Query(ctx, `SELECT surah, ayah FROM daily_ayahs ORDER BY position`)
	if err != nil {
		return nil, fmt.Errorf("select daily ayahs: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (AyahRef, error) {
		var a AyahRef
		err := row.Scan(&a.Surah, &a.Ayah)
		return a, err
	})
}

func (r *Repository) Reminders(ctx context.Context) ([]Reminder, error) {
	rows, err := r.db.Query(ctx, `SELECT id, texts FROM daily_reminders ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("select daily reminders: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Reminder, error) {
		var rm Reminder
		err := row.Scan(&rm.ID, &rm.Texts)
		return rm, err
	})
}

func (r *Repository) Overrides(ctx context.Context, from, to time.Time) ([]Override, error) {
	rows, err := r.db.Query(ctx, `
		SELECT date, surah, ayah, reminder_id
		FROM daily_content
		WHERE date BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return nil, fmt.Errorf("select daily content: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Override, error) {
		var o Override
		var surah, ayah, reminderID *int16
		if err := row.Scan(&o.Date, &surah, &ayah, &reminderID); err != nil {
			return o, err
		}
		if surah != nil && ayah != nil {
			o.Ayah = &AyahRef{Surah: int(*surah), Ayah: int(*ayah)}
		}
		if reminderID != nil {
			id := int(*reminderID)
			o.ReminderID = &id
		}
		return o, nil
	})
}
