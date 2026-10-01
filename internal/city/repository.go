package city

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("city not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const cityColumns = `c.id, c.country, c.lat, c.lon, c.time_zone,
	c.suggested_method, c.suggested_madhab, c.has_official_timetable, c.population`

// Get — один город по id.
func (r *Repository) Get(ctx context.Context, id string) (City, error) {
	var c City
	err := r.db.QueryRow(ctx, `SELECT `+cityColumns+` FROM cities c WHERE c.id = $1`, id).Scan(
		&c.ID, &c.Country, &c.Lat, &c.Lon, &c.TimeZone,
		&c.SuggestedMethod, &c.SuggestedMadhab, &c.HasOfficialTimetable, &c.Population,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return City{}, ErrNotFound
	}
	if err != nil {
		return City{}, fmt.Errorf("select city: %w", err)
	}

	cities := []City{c}
	if err := r.attachNames(ctx, cities); err != nil {
		return City{}, err
	}
	return cities[0], nil
}

// Search — поиск по названию на любом языке + фильтр по стране.
// Сортировка: сначала крупные города.
func (r *Repository) Search(ctx context.Context, p SearchParams) ([]City, error) {
	const q = `
		SELECT ` + cityColumns + `
		FROM cities c
		WHERE ($1 = '' OR EXISTS (
		          SELECT 1 FROM city_names n
		          WHERE n.city_id = c.id AND n.name_search LIKE '%' || $1 || '%'))
		  AND ($2 = '' OR c.country = $2)
		ORDER BY c.population DESC NULLS LAST, c.id
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, q, normalizeQuery(p.Query), p.Country, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("search cities: %w", err)
	}
	defer rows.Close()

	var cities []City
	for rows.Next() {
		var c City
		if err := rows.Scan(
			&c.ID, &c.Country, &c.Lat, &c.Lon, &c.TimeZone,
			&c.SuggestedMethod, &c.SuggestedMadhab, &c.HasOfficialTimetable, &c.Population,
		); err != nil {
			return nil, fmt.Errorf("scan city: %w", err)
		}
		cities = append(cities, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cities: %w", err)
	}

	if err := r.attachNames(ctx, cities); err != nil {
		return nil, err
	}
	return cities, nil
}

// Nearest — ближайший город и расстояние до него в км (формула гаверсинусов в SQL).
// Координаты НЕ логируем и НЕ сохраняем (§16 ТЗ).
func (r *Repository) Nearest(ctx context.Context, lat, lon float64) (City, float64, error) {
	const q = `
		SELECT ` + cityColumns + `,
		       6371 * 2 * asin(sqrt(
		           power(sin(radians($1 - c.lat) / 2), 2) +
		           cos(radians($1)) * cos(radians(c.lat)) *
		           power(sin(radians($2 - c.lon) / 2), 2)
		       )) AS distance_km
		FROM cities c
		ORDER BY distance_km
		LIMIT 1`

	var c City
	var dist float64
	err := r.db.QueryRow(ctx, q, lat, lon).Scan(
		&c.ID, &c.Country, &c.Lat, &c.Lon, &c.TimeZone,
		&c.SuggestedMethod, &c.SuggestedMadhab, &c.HasOfficialTimetable, &c.Population,
		&dist,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return City{}, 0, ErrNotFound
	}
	if err != nil {
		return City{}, 0, fmt.Errorf("nearest city: %w", err)
	}

	cities := []City{c}
	if err := r.attachNames(ctx, cities); err != nil {
		return City{}, 0, err
	}
	return cities[0], dist, nil
}

// attachNames загружает названия ОДНИМ запросом для всех городов сразу
// (а не по запросу на каждый город — это классическая проблема «N+1 запросов»).
func (r *Repository) attachNames(ctx context.Context, cities []City) error {
	if len(cities) == 0 {
		return nil
	}

	ids := make([]string, len(cities))
	index := make(map[string]int, len(cities))
	for i, c := range cities {
		ids[i] = c.ID
		index[c.ID] = i
		cities[i].Names = map[string]Name{}
	}

	rows, err := r.db.Query(ctx,
		`SELECT city_id, lang, name, COALESCE(region, '') FROM city_names WHERE city_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("select city names: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cityID, lang string
		var n Name
		if err := rows.Scan(&cityID, &lang, &n.Name, &n.Region); err != nil {
			return fmt.Errorf("scan city name: %w", err)
		}
		cities[index[cityID]].Names[lang] = n
	}
	return rows.Err()
}

// normalizeQuery приводит строку поиска к тому же виду, что и колонка name_search:
// нижний регистр + таджикские буквы → русские + экранирование спецсимволов LIKE.
var (
	tajikToRussian = strings.NewReplacer("ҳ", "х", "ҷ", "ч", "ӣ", "и", "ӯ", "у", "қ", "к", "ғ", "г", "ё", "е")
	likeEscaper    = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
)

func normalizeQuery(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	q = tajikToRussian.Replace(q)
	return likeEscaper.Replace(q)
}
