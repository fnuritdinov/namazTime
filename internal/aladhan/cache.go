package aladhan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service — Aladhan + кеш месяцев в базе (таблица aladhan_months).
// Для городов: сначала база, потом Aladhan. Для координат: только Aladhan, в базу НЕ пишем (§16).
type Service struct {
	Client *Client
	DB     *pgxpool.Pool
	Log    *slog.Logger
}

// Month возвращает дни месяца. cityID == "" — запрос по координатам.
func (s *Service) Month(ctx context.Context, cityID string, m Month) ([]Day, error) {
	if cityID == "" {
		// округляем до 0.01° (~1 км): точные координаты пользователя никуда не уходят
		m.Lat = math.Round(m.Lat*100) / 100
		m.Lon = math.Round(m.Lon*100) / 100
		return s.Client.FetchMonth(ctx, m)
	}

	if days, err := s.load(ctx, cityID, m); err == nil {
		return days, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.Log.Error("aladhan cache: load failed", "city", cityID, "err", err)
	}

	days, err := s.Client.FetchMonth(ctx, m)
	if err != nil {
		return nil, err
	}
	if err := s.save(ctx, cityID, m, days); err != nil {
		s.Log.Error("aladhan cache: save failed", "city", cityID, "err", err) // ответ всё равно отдаём
	}
	return days, nil
}

func (s *Service) load(ctx context.Context, cityID string, m Month) ([]Day, error) {
	var raw []byte
	err := s.DB.QueryRow(ctx, `
		SELECT days FROM aladhan_months
		WHERE city_id = $1 AND year = $2 AND month = $3 AND method = $4 AND school = $5`,
		cityID, m.Year, int(m.Month), m.Method, m.School,
	).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var days []Day
	if err := json.Unmarshal(raw, &days); err != nil {
		return nil, fmt.Errorf("decode cache: %w", err)
	}
	for i := range days { // в JSON время со сдвигом (+03:00) — возвращаем в пояс города
		d := &days[i]
		for _, t := range []*time.Time{&d.Date, &d.Imsak, &d.Fajr, &d.Sunrise, &d.Dhuhr, &d.Asr, &d.Maghrib, &d.Isha} {
			*t = t.In(m.Loc)
		}
	}
	return days, nil
}

func (s *Service) save(ctx context.Context, cityID string, m Month, days []Day) error {
	raw, err := json.Marshal(days)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO aladhan_months (city_id, year, month, method, school, days)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (city_id, year, month, method, school)
		DO UPDATE SET days = EXCLUDED.days, fetched_at = now()`,
		cityID, m.Year, int(m.Month), m.Method, m.School, string(raw),
	)
	return err
}
