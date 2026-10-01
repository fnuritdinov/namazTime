package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"nTime/internal/city"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

// GET /v1/cities?query=&country=&limit=&cursor=
func (s *Server) ListCities(ctx context.Context, req ListCitiesRequestObject) (ListCitiesResponseObject, error) {
	p := req.Params

	limit := defaultLimit
	if p.Limit != nil {
		limit = *p.Limit
	}
	if limit < 1 || limit > maxLimit {
		return ListCities400JSONResponse{BadRequestJSONResponse(
			newError(ctx, "invalid_argument", fmt.Sprintf("limit must be between 1 and %d", maxLimit)))}, nil
	}

	offset := 0
	if p.Cursor != nil {
		o, err := decodeCursor(*p.Cursor)
		if err != nil {
			return ListCities400JSONResponse{BadRequestJSONResponse(
				newError(ctx, "invalid_argument", "invalid cursor"))}, nil
		}
		offset = o
	}

	params := city.SearchParams{Limit: limit + 1, Offset: offset} // +1 — чтобы узнать, есть ли следующая страница
	if p.Query != nil {
		params.Query = *p.Query
	}
	if p.Country != nil {
		params.Country = strings.ToUpper(*p.Country)
	}

	cities, err := s.cities.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	var next *string
	if len(cities) > limit {
		cities = cities[:limit]
		c := encodeCursor(offset + limit)
		next = &c
	}

	items := make([]City, 0, len(cities)) // make, а не nil — иначе в JSON будет null вместо []
	for _, c := range cities {
		items = append(items, toAPICity(c))
	}
	return ListCities200JSONResponse{Items: items, NextCursor: next}, nil
}

// GET /v1/cities/nearest?lat=&lon=
// Координаты не логируем и не сохраняем (§16 ТЗ).
func (s *Server) GetNearestCity(ctx context.Context, req GetNearestCityRequestObject) (GetNearestCityResponseObject, error) {
	lat, lon := req.Params.Lat, req.Params.Lon
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return GetNearestCity400JSONResponse{BadRequestJSONResponse(
			newError(ctx, "invalid_argument", "lat must be in [-90, 90], lon in [-180, 180]"))}, nil
	}

	c, dist, err := s.cities.Nearest(ctx, lat, lon)
	if errors.Is(err, city.ErrNotFound) {
		return GetNearestCity404JSONResponse{NotFoundJSONResponse(
			newError(ctx, "not_found", "no cities available"))}, nil
	}
	if err != nil {
		return nil, err
	}

	a := toAPICity(c)
	return GetNearestCity200JSONResponse(NearestCity{
		Id:                   a.Id,
		Name:                 a.Name,
		Region:               a.Region,
		Country:              a.Country,
		Coordinate:           a.Coordinate,
		TimeZoneId:           a.TimeZoneId,
		SuggestedMethod:      a.SuggestedMethod,
		SuggestedMadhab:      a.SuggestedMadhab,
		HasOfficialTimetable: a.HasOfficialTimetable,
		Population:           a.Population,
		DistanceKm:           math.Round(dist*100) / 100,
	}), nil
}

// GET /v1/cities/{cityId}
func (s *Server) GetCity(ctx context.Context, req GetCityRequestObject) (GetCityResponseObject, error) {
	c, err := s.cities.Get(ctx, req.CityId)
	if errors.Is(err, city.ErrNotFound) {
		return GetCity404JSONResponse{NotFoundJSONResponse(
			newError(ctx, "not_found", fmt.Sprintf("city %q not found", req.CityId)))}, nil
	}
	if err != nil {
		return nil, err
	}
	return GetCity200JSONResponse(toAPICity(c)), nil
}

// ---------- преобразование типов ----------

func toAPICity(c city.City) City {
	names := map[string]string{}
	regions := map[string]string{}
	for lang, n := range c.Names {
		names[lang] = n.Name
		if n.Region != "" {
			regions[lang] = n.Region
		}
	}

	out := City{
		Id:                   c.ID,
		Name:                 toLocalized(names),
		Country:              c.Country,
		Coordinate:           Coordinate{Latitude: c.Lat, Longitude: c.Lon},
		TimeZoneId:           c.TimeZone,
		SuggestedMethod:      CalculationMethodId(c.SuggestedMethod),
		SuggestedMadhab:      Madhab(c.SuggestedMadhab),
		HasOfficialTimetable: c.HasOfficialTimetable,
		Population:           c.Population,
	}
	if len(regions) > 0 {
		r := toLocalized(regions)
		out.Region = &r
	}
	return out
}

// toLocalized превращает map {"ru": "...", "tg": "..."} в LocalizedString.
func toLocalized(m map[string]string) LocalizedString {
	var ls LocalizedString
	if v, ok := m["en"]; ok {
		ls.En = &v
	}
	if v, ok := m["ru"]; ok {
		ls.Ru = &v
	}
	if v, ok := m["tg"]; ok {
		ls.Tg = &v
	}
	if v, ok := m["ar"]; ok {
		ls.Ar = &v
	}
	return ls
}

// ---------- курсорная пагинация (§2.4 ТЗ) ----------
// Курсор «непрозрачный» для клиента: внутри просто смещение, закодированное в base64.
// Позже можно поменять внутренности (например, на keyset-пагинацию) — клиенты не заметят.

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(s string) (int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(string(b))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("bad cursor")
	}
	return n, nil
}
