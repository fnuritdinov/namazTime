package handler

import (
	"nTime/internal/device"
	"nTime/internal/hijrimonth"
	"nTime/internal/schedule"

	"nTime/internal/city"
)

type Server struct {
	cities   *city.Repository
	schedule *schedule.Service
	hijri    *hijrimonth.Service
	devices  *device.Service
	app      AppInfo
}

func NewServer(cityRepo *city.Repository, scheduleSvc *schedule.Service, hijriSvc *hijrimonth.Service,
	deviceSvc *device.Service, app AppInfo) *Server {
	return &Server{cities: cityRepo, schedule: scheduleSvc, hijri: hijriSvc, devices: deviceSvc, app: app}
}

var _ StrictServerInterface = (*Server)(nil)
