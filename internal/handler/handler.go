package handler

import (
	"nTime/internal/device"
	"nTime/internal/hijrimonth"
	"nTime/internal/schedule"
	"nTime/internal/sms"

	"nTime/internal/city"
)

type Server struct {
	cities   *city.Repository
	schedule *schedule.Service
	hijri    *hijrimonth.Service
	devices  *device.Service
	app      AppInfo
	sms      *sms.Service
}

func NewServer(cityRepo *city.Repository, scheduleSvc *schedule.Service, hijriSvc *hijrimonth.Service,
	deviceSvc *device.Service, app AppInfo) *Server {
	return &Server{cities: cityRepo, schedule: scheduleSvc, hijri: hijriSvc, devices: deviceSvc, app: app}
}

var _ StrictServerInterface = (*Server)(nil)

// WithSMS подключает отправку SMS (nil — выключено).
func (s *Server) WithSMS(svc *sms.Service) *Server {
	s.sms = svc
	return s
}
