package handler

import (
	"context"
	"errors"
	"fmt"

	"nTime/internal/qibla"
)

type Server struct{}

func New() *Server {
	return &Server{}
}

var _ StrictServerInterface = (*Server)(nil)

func (h *Server) GetQibla(ctx context.Context, req GetQiblaRequestObject) (GetQiblaResponseObject, error) {
	lat, lon := req.Params.Lat, req.Params.Lon

	if err := validateCoords(lat, lon); err != nil {
		return GetQibla400JSONResponse{BadRequestJSONResponse(newError("invalid coordinates", err.Error()))}, nil
	}

	return GetQibla200JSONResponse{
		Direction:  qibla.Direction(lat, lon),
		DistanceKm: qibla.Distance(lat, lon),
	}, nil
}

func (h *Server) GetPrayerTimes(ctx context.Context, req GetPrayerTimesRequestObject) (GetPrayerTimesResponseObject, error) {
	return nil, errors.New("not implement yet")
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
