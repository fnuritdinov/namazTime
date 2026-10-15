package shuro

import (
	"context"
	"errors"
	"log/slog"
	"time"
	_ "time/tzdata" // зона Asia/Dushanbe даже в alpine-образе без tzdata

	"nTime/internal/official"
)

// Fetcher — откуда берём таблицу (в main — *Client).
type Fetcher interface {
	FetchMonth(ctx context.Context, year int, month time.Month) ([]official.Row, error)
}

// Saver — куда сохраняем (в main — *Store).
type Saver interface {
	SaveMonth(ctx context.Context, cityID, sourceID string, rows []official.Row) error
}

// Worker раз в сутки в Hour:00 (по Loc) забирает с сайта Шуро текущий и следующий месяц.
// Ошибка сайта или разбора — только запись ERROR в журнал; старые данные в базе остаются.
type Worker struct {
	Fetch    Fetcher
	Save     Saver
	Log      *slog.Logger
	Loc      *time.Location // Asia/Dushanbe
	Hour     int            // 3 → 03:00
	CityID   string         // "dushanbe" — таблица публикуется для Душанбе
	SourceID string         // "shuroiulamo-tj"
}

// Run работает до отмены ctx: сразу при старте, потом каждый день в Hour:00.
func (w *Worker) Run(ctx context.Context) {
	w.SyncOnce(ctx)
	for {
		wait := time.Until(nextRun(time.Now(), w.Loc, w.Hour))
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			w.SyncOnce(ctx)
		}
	}
}

// SyncOnce забирает текущий и следующий месяц (по времени Душанбе).
func (w *Worker) SyncOnce(ctx context.Context) {
	now := time.Now().In(w.Loc)
	cur := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, w.Loc)
	for _, m := range []time.Time{cur, cur.AddDate(0, 1, 0)} {
		w.syncMonth(ctx, m.Year(), m.Month())
	}
}

func (w *Worker) syncMonth(ctx context.Context, year int, month time.Month) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	log := w.Log.With("year", year, "month", int(month))
	rows, err := w.Fetch.FetchMonth(ctx, year, month)
	switch {
	case errors.Is(err, ErrNotPublished):
		log.Info("shuro: timetable not published yet")
		return
	case err != nil:
		log.Error("shuro: fetch failed, keeping old data", "err", err)
		return
	}
	if err := w.Save.SaveMonth(ctx, w.CityID, w.SourceID, rows); err != nil {
		log.Error("shuro: save failed", "err", err)
		return
	}
	log.Info("shuro: timetable synced", "days", len(rows))
}

// nextRun — ближайшее время Hour:00 по loc строго после now.
func nextRun(now time.Time, loc *time.Location, hour int) time.Time {
	n := now.In(loc)
	t := time.Date(n.Year(), n.Month(), n.Day(), hour, 0, 0, 0, loc)
	if !t.After(n) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// проверка на этапе компиляции, что типы подходят к Worker
var (
	_ Fetcher = (*Client)(nil)
	_ Saver   = (*Store)(nil)
)
