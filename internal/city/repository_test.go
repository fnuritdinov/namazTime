package city

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"nTime/internal/storage"
)

// Unit-тест — без базы.
func TestNormalizeQuery(t *testing.T) {
	tests := map[string]string{
		"Хуҷанд":   "хучанд",
		"  Кӯлоб ": "кулоб",
		"Ҳисор":    "хисор",
		"50%":      `50\%`,
		"a_b":      `a\_b`,
	}
	for in, want := range tests {
		if got := normalizeQuery(in); got != want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

// Интеграционный тест — нужна база (TEST_DATABASE_URL).
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
	return NewRepository(pool)
}

func TestRepository(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		c, err := repo.Get(ctx, "khujand")
		if err != nil {
			t.Fatal(err)
		}
		if c.Names["tg"].Name != "Хуҷанд" || c.Names["ru"].Name != "Худжанд" {
			t.Errorf("names = %+v", c.Names)
		}
	})

	t.Run("Get unknown", func(t *testing.T) {
		if _, err := repo.Get(ctx, "atlantis"); !errors.Is(err, ErrNotFound) {
			t.Errorf("ожидали ErrNotFound, получили %v", err)
		}
	})

	t.Run("Search by tajik letters", func(t *testing.T) {
		cities, err := repo.Search(ctx, SearchParams{Query: "Хуҷ", Country: "TJ", Limit: 20})
		if err != nil {
			t.Fatal(err)
		}
		if len(cities) != 1 || cities[0].ID != "khujand" {
			t.Errorf("got %d cities: %+v", len(cities), cities)
		}
	})

	t.Run("Popular cities", func(t *testing.T) {
		cities, err := repo.Search(ctx, SearchParams{Country: "TJ", Limit: 3})
		if err != nil {
			t.Fatal(err)
		}
		if len(cities) != 3 || cities[0].ID != "dushanbe" {
			t.Errorf("первым должен быть Душанбе, got %+v", cities)
		}
	})

	t.Run("Nearest", func(t *testing.T) {
		c, dist, err := repo.Nearest(ctx, 38.56, 68.79)
		if err != nil {
			t.Fatal(err)
		}
		if c.ID != "dushanbe" || dist > 1 {
			t.Errorf("nearest = %s (%.2f км)", c.ID, dist)
		}
	})
}
