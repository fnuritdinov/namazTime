package takvim

import (
	"context"
	"log/slog"
	"time"

	"nTime/internal/official"
)

// Saver — куда сохранять дни (shuro.Store подходит: та же таблица официального времени).
type Saver interface {
	SaveMonth(ctx context.Context, cityID, sourceID string, rows []official.Row) error
}

// Worker раз в Every забирает 3 дня с takvim.tj и сохраняет их под своим источником.
// Если на ту же дату есть таблица Шуро, сервер отдаёт Шуро (см. official.Repository.Get).
type Worker struct {
	Client   *Client
	Save     Saver
	Log      *slog.Logger
	Every    time.Duration
	CityID   string // "dushanbe"
	SourceID string // "takvim-tj"
}

func (w *Worker) Run(ctx context.Context) {
	w.SyncOnce(ctx)
	t := time.NewTicker(w.Every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.SyncOnce(ctx)
		}
	}
}

func (w *Worker) SyncOnce(ctx context.Context) {
	rows, err := w.Client.FetchDays(ctx)
	if err != nil {
		w.Log.Error("takvim: fetch failed", "err", err)
		return
	}
	// Таблица хранится по годам: 3 дня могут попасть на 31 декабря и 1–2 января
	byYear := map[int][]official.Row{}
	for _, r := range rows {
		byYear[r.Date.Year()] = append(byYear[r.Date.Year()], r)
	}
	for year, part := range byYear {
		if err := w.Save.SaveMonth(ctx, w.CityID, w.SourceID, part); err != nil {
			w.Log.Error("takvim: save failed", "year", year, "err", err)
			return
		}
	}
	w.Log.Info("takvim: days synced",
		"from", rows[0].Date.Format("2006-01-02"), "to", rows[len(rows)-1].Date.Format("2006-01-02"))
}
