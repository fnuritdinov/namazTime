package handler

import (
	"context"
	"errors"

	"nTime/internal/geo"
	"nTime/internal/prayer"
)

type Server struct {
	prayer *prayer.Service
	geo    *geo.Locator
}

func NewServer(prayerSvc *prayer.Service, geoLoc *geo.Locator) *Server {
	return &Server{prayer: prayerSvc, geo: geoLoc}
}

// Проверка при компиляции: Server реализует все методы из openapi.yaml.
var _ StrictServerInterface = (*Server)(nil)

var errNotImplemented = errors.New("not implemented yet")

// GET /v1/config — шаг E
func (s *Server) GetConfig(ctx context.Context, req GetConfigRequestObject) (GetConfigResponseObject, error) {
	return nil, errNotImplemented
}

// GET /v1/cities — шаг B
func (s *Server) ListCities(ctx context.Context, req ListCitiesRequestObject) (ListCitiesResponseObject, error) {
	return nil, errNotImplemented
}

// GET /v1/cities/nearest — шаг B
func (s *Server) GetNearestCity(ctx context.Context, req GetNearestCityRequestObject) (GetNearestCityResponseObject, error) {
	return nil, errNotImplemented
}

// GET /v1/cities/{cityId} — шаг B
func (s *Server) GetCity(ctx context.Context, req GetCityRequestObject) (GetCityResponseObject, error) {
	return nil, errNotImplemented
}

// GET /v1/calculation-methods — шаг C
func (s *Server) ListCalculationMethods(ctx context.Context, req ListCalculationMethodsRequestObject) (ListCalculationMethodsResponseObject, error) {
	return nil, errNotImplemented
}

// GET /v1/prayer-times — шаг D
func (s *Server) GetPrayerTimes(ctx context.Context, req GetPrayerTimesRequestObject) (GetPrayerTimesResponseObject, error) {
	return nil, errNotImplemented
}
