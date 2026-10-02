package handler

import (
	"nTime/internal/hijrimonth"
	"nTime/internal/schedule"

	"nTime/internal/city"
)

type Server struct {
	cities   *city.Repository
	schedule *schedule.Service
	hijri    *hijrimonth.Service
	app      AppInfo
}

func NewServer(cityRepo *city.Repository, scheduleSvc *schedule.Service, hijriSvc *hijrimonth.Service, app AppInfo) *Server {
	return &Server{cities: cityRepo, schedule: scheduleSvc, hijri: hijriSvc, app: app}
}

var _ StrictServerInterface = (*Server)(nil)
