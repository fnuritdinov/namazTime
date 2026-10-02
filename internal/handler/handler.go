package handler

import (
	"nTime/internal/schedule"

	"nTime/internal/city"
)

type Server struct {
	cities   *city.Repository
	schedule *schedule.Service
	app      AppInfo
}

func NewServer(cityRepo *city.Repository, scheduleSvc *schedule.Service, app AppInfo) *Server {
	return &Server{cities: cityRepo, schedule: scheduleSvc, app: app}
}

var _ StrictServerInterface = (*Server)(nil)
