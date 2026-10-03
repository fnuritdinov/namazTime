package device

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type fakeCleaner struct {
	mu      sync.Mutex
	befores []time.Time
}

func (f *fakeCleaner) DeleteInactive(_ context.Context, before time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.befores = append(f.befores, before)
	return 1, nil
}

func (f *fakeCleaner) calls() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Time(nil), f.befores...)
}

func TestRunCleanup(t *testing.T) {
	c := &fakeCleaner{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunCleanup(ctx, c, 12, 20*time.Millisecond, slog.New(slog.DiscardHandler))
		close(done)
	}()

	// Ждём первый запуск + хотя бы один тик. Не фиксированный sleep, а «пока не случится»,
	// с пределом: на медленной машине таймеры срабатывают позже.
	deadline := time.Now().Add(2 * time.Second)
	for len(c.calls()) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case <-done: // остановился по ctx
	case <-time.After(time.Second):
		t.Fatal("RunCleanup did not stop after cancel")
	}

	calls := c.calls()
	if len(calls) < 2 {
		t.Fatalf("expected first run at start + ticks, got %d calls", len(calls))
	}
	// Граница — 12 месяцев назад (± день на всякий случай)
	want := time.Now().AddDate(-1, 0, 0)
	if d := calls[0].Sub(want); d > 24*time.Hour || d < -24*time.Hour {
		t.Errorf("before = %v, want about %v", calls[0], want)
	}
}
