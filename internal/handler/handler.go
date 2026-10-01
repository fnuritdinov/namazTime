package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"nTime/internal/prayer"
	"nTime/internal/qibla"
)

type Server struct {
	prayer *prayer.Service
}

func NewServer(prayerSvc *prayer.Service) *Server {
	return &Server{prayer: prayerSvc}
}

var _ StrictServerInterface = (*Server)(nil)

func (s *Server) GetPrayerTimes(ctx context.Context, req GetPrayerTimesRequestObject) (GetPrayerTimesResponseObject, error) {
	p := req.Params
	if err := validateCoords(p.Lat, p.Lon); err != nil {
		return GetPrayerTimes400JSONResponse{BadRequestJSONResponse(newError("invalid_coordinates", err.Error()))}, nil
	}

	// Дата: из запроса или сегодня (UTC)
	date := time.Now().UTC()
	if p.Date != nil {
		date = p.Date.Time
	}
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	// Мазхаб: по умолчанию ханафитский
	school := 1
	if p.School != nil {
		school = int(*p.School)
	}

	d, err := s.prayer.GetTimings(ctx, p.Lat, p.Lon, date, school)
	if errors.Is(err, prayer.ErrUpstream) {
		return GetPrayerTimes502JSONResponse{BadGatewayJSONResponse(newError("upstream_unavailable", "prayer times provider is unavailable"))}, nil
	}
	if err != nil {
		return nil, err // → 500, детали уйдут в лог
	}

	return GetPrayerTimes200JSONResponse(toPrayerTimesResponse(p.Lat, p.Lon, d)), nil
}

// toPrayerTimesResponse переводит наши типы в типы API.
func toPrayerTimesResponse(lat, lon float64, d prayer.DayTimings) PrayerTimesResponse {
	t := d.Timings
	resp := PrayerTimesResponse{
		Date:     openapi_types.Date{Time: d.Date},
		Location: Location{Lat: lat, Lon: lon, Country: ""}, // страну определим на шаге 6
		Source:   PrayerTimesResponseSource(d.Source),
		Timings: Timings{
			Fajr:    PrayerTime{Start: t.Fajr},
			Sunrise: PrayerTime{Start: t.Sunrise},
			Dhuhr:   PrayerTime{Start: t.Dhuhr},
			Asr:     PrayerTime{Start: t.Asr},
			Maghrib: PrayerTime{Start: t.Maghrib},
			Isha:    PrayerTime{Start: t.Isha},
		},
	}
	if d.HijriDate != "" {
		resp.HijriDate = &d.HijriDate
	}
	return resp
}

func (s *Server) GetQibla(ctx context.Context, req GetQiblaRequestObject) (GetQiblaResponseObject, error) {
	lat, lon := req.Params.Lat, req.Params.Lon
	if err := validateCoords(lat, lon); err != nil {
		return GetQibla400JSONResponse{BadRequestJSONResponse(newError("invalid_coordinates", err.Error()))}, nil
	}
	return GetQibla200JSONResponse{
		Direction:  qibla.Direction(lat, lon),
		DistanceKm: qibla.Distance(lat, lon),
	}, nil
}

func validateCoords(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("lat must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("lon must be between -180 and 180")
	}
	return nil
}

func newError(code, msg string) Error {
	return Error{Error: ErrorBody{Code: code, Message: msg}}
}
