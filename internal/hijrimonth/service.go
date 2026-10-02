package hijrimonth

import (
	"context"
	"fmt"
	"regexp"
)

// InvalidError — неверный параметр (→ 400).
type InvalidError struct{ Msg string }

func (e *InvalidError) Error() string { return e.Msg }

// Store — где хранятся месяцы и дуа (в main — Repository, в тестах — заглушка).
type Store interface {
	Months(ctx context.Context, country string, hijriYear int) ([]Month, error)
	Duas(ctx context.Context, category string) ([]Dua, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

var countryRe = regexp.MustCompile(`^[A-Z]{2}$`)

func validate(country string, year int) error {
	if !countryRe.MatchString(country) {
		return &InvalidError{Msg: "country must be ISO 3166-1 alpha-2, e.g. TJ"}
	}
	if year < 1400 || year > 1600 {
		return &InvalidError{Msg: "hijri year must be between 1400 and 1600"}
	}
	return nil
}

// Months — известные начала месяцев (может быть меньше 12).
func (s *Service) Months(ctx context.Context, country string, year int) ([]Month, error) {
	if err := validate(country, year); err != nil {
		return nil, err
	}
	return s.store.Months(ctx, country, year)
}

// Ramadan собирает даты Рамадана из начал 9-го и 10-го месяцев.
// Нет 9-го месяца → ErrNotFound (по ТЗ приложение тогда использует Умм аль-Кура).
func (s *Service) Ramadan(ctx context.Context, country string, year int) (Ramadan, error) {
	months, err := s.Months(ctx, country, year)
	if err != nil {
		return Ramadan{}, err
	}

	var ramadan, shawwal *Month
	for i := range months {
		switch months[i].Month {
		case 9:
			ramadan = &months[i]
		case 10:
			shawwal = &months[i]
		}
	}
	if ramadan == nil {
		return Ramadan{}, fmt.Errorf("%w: no Ramadan %d for %s", ErrNotFound, year, country)
	}

	// Ид = 1 Шавваля. Если Шавваль ещё не объявлен — ожидаем 30 дней поста.
	eid := ramadan.Start.AddDate(0, 0, 30)
	if shawwal != nil {
		eid = shawwal.Start
	}
	length := int(eid.Sub(ramadan.Start).Hours() / 24)

	duas, err := s.store.Duas(ctx, "ramadan")
	if err != nil {
		return Ramadan{}, err
	}

	return Ramadan{
		HijriYear:    year,
		Country:      country,
		Status:       ramadan.Status,
		Start:        ramadan.Start,
		End:          eid.AddDate(0, 0, -1),
		Length:       length,
		EidAlFitr:    eid,
		LaylatAlQadr: ramadan.Start.AddDate(0, 0, 25), // ночь на 27-е: вечер 26-го дня
		ConfirmedAt:  ramadan.ConfirmedAt,
		SourceName:   ramadan.SourceName,
		Duas:         duas,
	}, nil
}
