// Package daily — контент дня (§10 ТЗ): аят дня и напоминание дня.
//
// По умолчанию работает ротация: день № N от 1970-01-01 берёт аят № N по кругу
// и напоминание № N по кругу. Поэтому у всех пользователей в один день одно и то же,
// и ответ можно кешировать. Редактор может задать свой аят или напоминание на дату.
package daily

import (
	"context"
	"fmt"
	"time"
)

const maxDays = 31

// InvalidError — неверный параметр (→ 400).
type InvalidError struct{ Msg string }

func (e *InvalidError) Error() string { return e.Msg }

// AyahRef — ссылка на аят: текст Корана хранит приложение.
type AyahRef struct{ Surah, Ayah int }

// Reminder — напоминание на нескольких языках.
type Reminder struct {
	ID    int
	Texts map[string]string // язык → текст
}

// Override — редакционный выбор на дату. nil — взять из ротации.
type Override struct {
	Date       time.Time
	Ayah       *AyahRef
	ReminderID *int
}

// Day — контент одного дня.
type Day struct {
	Date     time.Time
	Ayah     *AyahRef          // nil — список аятов пуст
	Reminder map[string]string // nil — напоминаний нет
}

// Store — где лежит контент (в main — Repository, в тестах — заглушка).
type Store interface {
	Ayahs(ctx context.Context) ([]AyahRef, error)
	Reminders(ctx context.Context) ([]Reminder, error)
	Overrides(ctx context.Context, from, to time.Time) ([]Override, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Days — контент на каждый день из [from, to] (не больше 31 дня).
func (s *Service) Days(ctx context.Context, from, to time.Time) ([]Day, error) {
	from, to = civil(from), civil(to)
	if to.Before(from) {
		return nil, &InvalidError{Msg: "to must not be before from"}
	}
	if n := dayNumber(to) - dayNumber(from) + 1; n > maxDays {
		return nil, &InvalidError{Msg: fmt.Sprintf("date range must not exceed %d days, got %d", maxDays, n)}
	}

	ayahs, err := s.store.Ayahs(ctx)
	if err != nil {
		return nil, err
	}
	reminders, err := s.store.Reminders(ctx)
	if err != nil {
		return nil, err
	}
	overrides, err := s.store.Overrides(ctx, from, to)
	if err != nil {
		return nil, err
	}

	byDate := map[string]Override{}
	for _, o := range overrides {
		byDate[civil(o.Date).Format(time.DateOnly)] = o
	}
	reminderByID := map[int]map[string]string{}
	for _, r := range reminders {
		reminderByID[r.ID] = r.Texts
	}

	var days []Day
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		n := dayNumber(d)
		day := Day{Date: d}

		// Ротация
		if len(ayahs) > 0 {
			a := ayahs[n%len(ayahs)]
			day.Ayah = &a
		}
		if len(reminders) > 0 {
			day.Reminder = reminders[n%len(reminders)].Texts
		}

		// Выбор редактора важнее ротации
		if o, ok := byDate[d.Format(time.DateOnly)]; ok {
			if o.Ayah != nil {
				day.Ayah = o.Ayah
			}
			if o.ReminderID != nil {
				if texts, ok := reminderByID[*o.ReminderID]; ok {
					day.Reminder = texts
				}
			}
		}
		days = append(days, day)
	}
	return days, nil
}

// dayNumber — сколько дней прошло с 1970-01-01 (для ротации по кругу).
func dayNumber(t time.Time) int {
	return int(civil(t).Unix() / 86400)
}

// civil — полночь UTC того же календарного дня.
func civil(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
