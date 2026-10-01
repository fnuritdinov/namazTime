package handler

import (
	"context"
	"errors"
	"nTime/internal/schedule"

	"nTime/internal/city"
)

type Server struct {
	cities   *city.Repository
	schedule *schedule.Service
}

func NewServer(cityRepo *city.Repository, scheduleSvc *schedule.Service) *Server {
	return &Server{cities: cityRepo, schedule: scheduleSvc}
}

var _ StrictServerInterface = (*Server)(nil)

var errNotImplemented = errors.New("not implemented yet")

// GET /v1/config — шаг E
func (s *Server) GetConfig(ctx context.Context, req GetConfigRequestObject) (GetConfigResponseObject, error) {
	return nil, errNotImplemented
}
