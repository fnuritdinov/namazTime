package handler

import (
	"nTime/internal/daily"
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
	daily    *daily.Service
	app      AppInfo
}

func NewServer(cityRepo *city.Repository, scheduleSvc *schedule.Service, hijriSvc *hijrimonth.Service,
	deviceSvc *device.Service, dailySvc *daily.Service, app AppInfo) *Server {
	return &Server{cities: cityRepo, schedule: scheduleSvc, hijri: hijriSvc, devices: deviceSvc, daily: dailySvc, app: app}
}

var _ StrictServerInterface = (*Server)(nil)
