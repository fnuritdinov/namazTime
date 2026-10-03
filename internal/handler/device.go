package handler

import (
	"context"
	"errors"

	"nTime/internal/device"
)

// POST /v1/devices
func (s *Server) RegisterDevice(ctx context.Context, req RegisterDeviceRequestObject) (RegisterDeviceResponseObject, error) {
	if req.Body == nil {
		return RegisterDevice400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", "request body is required"))}, nil
	}
	b := req.Body
	id, token, err := s.devices.Register(ctx, device.Device{
		Platform:   string(b.Platform),
		AppVersion: b.AppVersion,
		Locale:     deref(b.Locale),
		TimeZoneID: deref(b.TimeZoneId),
		Country:    deref(b.Country),
	})

	var ie *device.InvalidError
	if errors.As(err, &ie) {
		return RegisterDevice400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", ie.Msg))}, nil
	}
	if err != nil {
		return nil, err
	}
	return RegisterDevice201JSONResponse{DeviceId: id, Token: token}, nil
}

// PUT /v1/devices/me/push — устройство уже проверено middleware device.Auth
func (s *Server) SetDevicePush(ctx context.Context, req SetDevicePushRequestObject) (SetDevicePushResponseObject, error) {
	if req.Body == nil {
		return SetDevicePush400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", "request body is required"))}, nil
	}
	err := s.devices.SetPush(ctx, device.IDFrom(ctx), device.Push{
		APNsToken:   req.Body.ApnsToken,
		Environment: string(req.Body.Environment),
		Topics:      req.Body.Topics,
	})

	var ie *device.InvalidError
	if errors.As(err, &ie) {
		return SetDevicePush400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", ie.Msg))}, nil
	}
	if err != nil {
		return nil, err
	}
	return SetDevicePush204Response{}, nil
}

// deref — значение необязательного поля или "".
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
