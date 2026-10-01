package prayer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

// --- заглушки ---

type fakeStore struct {
	data map[Key]DayTimings
}

func (f *fakeStore) Get(_ context.Context, k Key) (DayTimings, error) {
	d, ok := f.data[k]
	if !ok {
		return DayTimings{}, ErrNotFound
	}
	return d, nil
}

func (f *fakeStore) Save(_ context.Context, k Key, d DayTimings) error {
	f.data[k] = d
	return nil
}

type fakeProvider struct {
	calls int
	err   error
}

func (f *fakeProvider) Fetch(_ context.Context, _, _ float64, date time.Time, _, _ int) (DayTimings, error) {
	f.calls++
	if f.err != nil {
		return DayTimings{}, f.err
	}
	return DayTimings{Date: date, Source: "aladhan", Timings: Timings{Fajr: "04:52"}}, nil
}

func newTestService(p *fakeProvider) *Service {
	log := slog.New(slog.NewTextHandler(io.Discard, nil)) // логи в тестах не нужны
	return NewService(&fakeStore{data: map[Key]DayTimings{}}, p, 3, log)
}

// --- тесты ---

func TestService_CachesInStore(t *testing.T) {
	p := &fakeProvider{}
	svc := newTestService(p)
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	// Первый запрос → идём в провайдер
	if _, err := svc.GetTimings(context.Background(), 41.3111, 69.2797, date, 1); err != nil {
		t.Fatal(err)
	}
	// Второй запрос рядом (те же координаты после округления) → берём из хранилища
	if _, err := svc.GetTimings(context.Background(), 41.3149, 69.2801, date, 1); err != nil {
		t.Fatal(err)
	}

	if p.calls != 1 {
		t.Errorf("провайдер вызван %d раз, ожидали 1", p.calls)
	}
}

func TestService_ProviderError(t *testing.T) {
	p := &fakeProvider{err: errors.New("timeout")}
	svc := newTestService(p)

	_, err := svc.GetTimings(context.Background(), 41.31, 69.24, time.Now(), 1)
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("ожидали ErrUpstream, получили %v", err)
	}
}
