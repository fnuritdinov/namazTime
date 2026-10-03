package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// serve запускает сервер и ждёт: либо сигнал остановки (ctx), либо ошибку запуска.
// При остановке даёт текущим запросам до 10 секунд завершиться.
func serve(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
