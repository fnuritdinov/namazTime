package device

import (
	"context"
	"log/slog"
	"time"
)

// Cleaner удаляет устройства, которые не заходили с момента before (в main — Repository).
type Cleaner interface {
	DeleteInactive(ctx context.Context, before time.Time) (int64, error)
}

// RunCleanup раз в every удаляет устройства, неактивные дольше months месяцев (§16 ТЗ).
// Первый проход — сразу при старте. Работает, пока не отменён ctx (остановка сервера).
func RunCleanup(ctx context.Context, c Cleaner, months int, every time.Duration, log *slog.Logger) {
	run := func() {
		n, err := c.DeleteInactive(ctx, time.Now().AddDate(0, -months, 0))
		switch {
		case err != nil && ctx.Err() == nil:
			log.Warn("device cleanup failed", "err", err)
		case n > 0:
			log.Info("deleted inactive devices", "count", n, "inactiveMonths", months)
		}
	}

	run()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
