package handler

import (
	"context"
	"errors"

	"nTime/internal/city"
	"nTime/internal/geo"
	"nTime/internal/prayer"
)

type Server struct {
	prayer *prayer.Service
	geo    *geo.Locator
	cities *city.Repository
}

func NewServer(prayerSvc *prayer.Service, geoLoc *geo.Locator, cityRepo *city.Repository) *Server {
	return &Server{prayer: prayerSvc, geo: geoLoc, cities: cityRepo}
}

var _ StrictServerInterface = (*Server)(nil)

var errNotImplemented = errors.New("not implemented yet")

// GET /v1/config — шаг E
func (s *Server) GetConfig(ctx context.Context, req GetConfigRequestObject) (GetConfigResponseObject, error) {
	return nil, errNotImplemented
}

// GET /v1/prayer-times — шаг D
func (s *Server) GetPrayerTimes(ctx context.Context, req GetPrayerTimesRequestObject) (GetPrayerTimesResponseObject, error) {
	return nil, errNotImplemented
}
