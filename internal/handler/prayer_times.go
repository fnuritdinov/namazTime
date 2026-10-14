package handler

import (
	"context"
	"errors"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"nTime/internal/city"
	"nTime/internal/schedule"
)

// GET /v1/prayer-times
func (s *Server) GetPrayerTimes(ctx context.Context, req GetPrayerTimesRequestObject) (GetPrayerTimesResponseObject, error) {
	p := req.Params
	r := schedule.Request{
		Lat:  p.Lat,
		Lon:  p.Lon,
		From: p.From.Time,
		To:   p.To.Time,
	}
	if p.CityId != nil {
		r.CityID = *p.CityId
	}
	if p.TimeZoneId != nil {
		r.TimeZoneID = *p.TimeZoneId
	}
	if p.Method != nil {
		r.Method = string(*p.Method)
	}
	if p.Madhab != nil {
		r.Madhab = string(*p.Madhab)
	}
	if p.Source != nil {
		r.Source = string(*p.Source)
	}

	sch, err := s.schedule.Get(ctx, r)

	var ie *schedule.InvalidError
	switch {
	case errors.As(err, &ie):
		return GetPrayerTimes400JSONResponse{BadRequestJSONResponse(
			newError(ctx, "invalid_argument", ie.Msg))}, nil
	case errors.Is(err, city.ErrNotFound):
		return GetPrayerTimes404JSONResponse{NotFoundJSONResponse(
			newError(ctx, "not_found", fmt.Sprintf("city %q not found", r.CityID)))}, nil
	case errors.Is(err, schedule.ErrUnsupported):
		return GetPrayerTimes422JSONResponse{UnsupportedJSONResponse(
			newError(ctx, "unsupported", err.Error()))}, nil
	case err != nil:
		return nil, err // → 500
	}

	return GetPrayerTimes200JSONResponse(toPrayerTimesResponse(sch)), nil
}

func toPrayerTimesResponse(s schedule.Schedule) PrayerTimesResponse {
	rng := func(r schedule.Range) TimeRange { return TimeRange{Start: r.Start, End: r.End} }

	days := make([]PrayerDay, 0, len(s.Days))
	for _, d := range s.Days {
		monthName := toLocalized(d.Hijri.MonthName())
		imsak := d.Times.Imsak

		times := PrayerPeriods{
			Fajr:    rng(d.Periods.Fajr),
			Dhuhr:   rng(d.Periods.Dhuhr),
			Asr:     rng(d.Periods.Asr),
			Maghrib: rng(d.Periods.Maghrib),
			Isha:    rng(d.Periods.Isha),
		}
		if d.Periods.Makruh != nil {
			mk := rng(*d.Periods.Makruh)
			times.Makruh = &mk
		}

		days = append(days, PrayerDay{
			Date: openapi_types.Date{Time: d.Date},
			Hijri: HijriDate{
				Day:       d.Hijri.Day,
				Month:     d.Hijri.Month,
				Year:      d.Hijri.Year,
				MonthName: &monthName,
			},
			Imsak:   &imsak,
			Sunrise: d.Times.Sunrise,
			Times:   times,
		})
	}

	resp := PrayerTimesResponse{
		TimeZoneId: s.TimeZoneID,
		Method:     CalculationMethodId(s.Method),
		Madhab:     Madhab(s.Madhab),
		Source:     PrayerSource(s.Source),
		Days:       days,
	}
	if s.CityID != "" {
		id := s.CityID
		resp.CityId = &id
	}

	// Название источника и дата обновления — только для официального расписания
	if len(s.SourceName) > 0 {
		name := toLocalized(s.SourceName)
		resp.SourceName = &name
	}
	resp.UpdatedAt = s.UpdatedAt

	return resp
}
