package prayer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// ErrUpstream — внешний источник (Aladhan) недоступен или ответил ошибкой.
var ErrUpstream = errors.New("prayer times provider unavailable")

// Store — где храним время намаза (сейчас Postgres).
type Store interface {
	Get(ctx context.Context, k Key) (DayTimings, error)
	Save(ctx context.Context, k Key, d DayTimings) error
}

// Provider — откуда берём время намаза (сейчас Aladhan, позже Шурои уламо).
type Provider interface {
	Fetch(ctx context.Context, lat, lon float64, date time.Time, method, school int) (DayTimings, error)
}

type Service struct {
	store    Store
	provider Provider
	method   int
	log      *slog.Logger
}

func NewService(store Store, provider Provider, method int, log *slog.Logger) *Service {
	return &Service{store: store, provider: provider, method: method, log: log}
}

func (s *Service) GetTimings(ctx context.Context, lat, lon float64, date time.Time, school int) (DayTimings, error) {
	key := Key{Lat: lat, Lon: lon, Date: date, Method: s.method, School: school}.Normalize()

	// 1. Ищем в БД
	d, err := s.store.Get(ctx, key)
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, ErrNotFound) {
		// БД сломалась — не падаем, а пробуем внешний источник
		s.log.Warn("prayer store get failed", "err", err)
	}

	// 2. Идём в Aladhan
	s.log.Info("fetching prayer times from provider", "date", key.Date.Format("2006-01-02"))
	d, err = s.provider.Fetch(ctx, key.Lat, key.Lon, key.Date, key.Method, key.School)
	if err != nil {
		s.log.Error("provider fetch failed", "err", err)
		return DayTimings{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	// 3. Сохраняем. Если не получилось — всё равно отдаём пользователю ответ
	if err := s.store.Save(ctx, key, d); err != nil {
		s.log.Warn("prayer store save failed", "err", err)
	}
	return d, nil
}
