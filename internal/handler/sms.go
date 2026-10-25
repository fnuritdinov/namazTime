package handler

import (
	"context"
	"errors"
	"time"

	"nTime/internal/city"
	"nTime/internal/device"
	"nTime/internal/schedule"
	"nTime/internal/sms"
)

// POST /v1/devices/me/sms — время намаза другу по SMS. Устройство уже проверено middleware device.Auth.
func (s *Server) SendPrayerSms(ctx context.Context, req SendPrayerSmsRequestObject) (SendPrayerSmsResponseObject, error) {
	if s.sms == nil {
		return SendPrayerSms503JSONResponse{UnavailableJSONResponse(newError(ctx, "unavailable", "sms is not configured"))}, nil
	}
	if req.Body == nil {
		return SendPrayerSms400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", "request body is required"))}, nil
	}
	b := req.Body

	name, err := sms.CleanName(deref(b.Name))
	if err != nil {
		return SendPrayerSms400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", err.Error()))}, nil
	}

	// «Сегодня» — по поясу пользователя; для городов Таджикистана — по Душанбе
	tz := "Asia/Dushanbe"
	if b.TimeZoneId != nil && *b.TimeZoneId != "" {
		tz = *b.TimeZoneId
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return SendPrayerSms400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", "unknown timeZoneId"))}, nil
	}
	now := time.Now().In(loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	sch, err := s.schedule.Get(ctx, schedule.Request{
		CityID: deref(b.CityId), Lat: b.Lat, Lon: b.Lon, TimeZoneID: deref(b.TimeZoneId),
		From: day, To: day,
	})
	var ie *schedule.InvalidError
	switch {
	case errors.As(err, &ie):
		return SendPrayerSms400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", ie.Msg))}, nil
	case errors.Is(err, city.ErrNotFound):
		return SendPrayerSms404JSONResponse{NotFoundJSONResponse(newError(ctx, "not_found", "city not found"))}, nil
	case errors.Is(err, schedule.ErrUnsupported):
		return SendPrayerSms422JSONResponse{UnsupportedJSONResponse(newError(ctx, "unsupported", err.Error()))}, nil
	case errors.Is(err, schedule.ErrUnavailable):
		return SendPrayerSms503JSONResponse{UnavailableJSONResponse(newError(ctx, "unavailable", "prayer times provider is temporarily unavailable"))}, nil
	case err != nil:
		return nil, err
	}
	if len(sch.Days) == 0 {
		return SendPrayerSms422JSONResponse{UnsupportedJSONResponse(newError(ctx, "unsupported", "no prayer times for today"))}, nil
	}

	d := sch.Days[0]
	lang := "tg"
	if b.Lang != nil {
		lang = string(*b.Lang)
	}
	text := sms.PrayerText(lang, name, sms.Prayers{
		Date: d.Date, Fajr: d.Times.Fajr, Dhuhr: d.Times.Dhuhr, Asr: d.Times.Asr,
		Maghrib: d.Times.Maghrib, Isha: d.Times.Isha,
	})

	remaining, err := s.sms.Send(ctx, device.IDFrom(ctx), b.Phone, text)
	switch {
	case errors.Is(err, sms.ErrBadPhone):
		return SendPrayerSms400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", err.Error()))}, nil
	case errors.Is(err, sms.ErrDeviceLimit), errors.Is(err, sms.ErrPhoneLimit), errors.Is(err, sms.ErrGlobalLimit):
		return SendPrayerSms429JSONResponse(newError(ctx, "sms_limit", err.Error())), nil
	case errors.Is(err, sms.ErrUnavailable):
		return SendPrayerSms503JSONResponse{UnavailableJSONResponse(newError(ctx, "unavailable", "sms service is temporarily unavailable"))}, nil
	case err != nil:
		return nil, err
	}
	return SendPrayerSms200JSONResponse{RemainingToday: remaining}, nil
}
