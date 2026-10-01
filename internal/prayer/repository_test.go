package prayer

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nTime/internal/storage"
)

func newTestRepo(t *testing.T) *Repository {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан — пропускаем интеграционный тест")
	}
	if err := storage.Migrate(url); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// Тестовая дата далеко в будущем, чтобы не пересечься с реальными данными
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM prayer_times WHERE date = '2099-01-01'`)
	})

	return NewRepository(pool)
}

func TestRepository_SaveAndGet(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	key := Key{
		Lat: 41.3111, Lon: 69.2797,
		Date:   time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		Method: 3, School: 1,
	}.Normalize()

	// 1. Записи ещё нет
	if _, err := repo.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ожидали ErrNotFound, получили %v", err)
	}

	// 2. Сохраняем
	want := DayTimings{
		Date: key.Date, HijriDate: "01-01-1521", Source: "aladhan",
		Timings: Timings{"05:00", "06:30", "12:30", "15:30", "17:45", "19:15"},
	}
	if err := repo.Save(ctx, key, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 3. Повторное сохранение — без ошибки (ON CONFLICT DO NOTHING)
	if err := repo.Save(ctx, key, want); err != nil {
		t.Fatalf("повторный Save: %v", err)
	}

	// 4. Читаем и сравниваем
	got, err := repo.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Timings != want.Timings {
		t.Errorf("Timings = %+v, want %+v", got.Timings, want.Timings)
	}
	if got.Source != want.Source || got.HijriDate != want.HijriDate {
		t.Errorf("Source/Hijri = %q/%q", got.Source, got.HijriDate)
	}
}
