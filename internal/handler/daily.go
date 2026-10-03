package handler

import (
	"context"
	"errors"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"nTime/internal/daily"
)

// GET /v1/daily?date=2026-10-03  или  ?from=2026-10-03&to=2026-10-09
func (s *Server) GetDaily(ctx context.Context, req GetDailyRequestObject) (GetDailyResponseObject, error) {
	p := req.Params
	if p.Date != nil && (p.From != nil || p.To != nil) {
		return GetDaily400JSONResponse{BadRequestJSONResponse(
			newError(ctx, "invalid_argument", "use either date or from/to"))}, nil
	}

	// По умолчанию — сегодня (UTC); to по умолчанию = from
	from := time.Now().UTC()
	switch {
	case p.Date != nil:
		from = p.Date.Time
	case p.From != nil:
		from = p.From.Time
	}
	to := from
	if p.To != nil {
		to = p.To.Time
	}

	days, err := s.daily.Days(ctx, from, to)
	var ie *daily.InvalidError
	if errors.As(err, &ie) {
		return GetDaily400JSONResponse{BadRequestJSONResponse(newError(ctx, "invalid_argument", ie.Msg))}, nil
	}
	if err != nil {
		return nil, err
	}

	items := make([]Daily, 0, len(days))
	for _, d := range days {
		item := Daily{Date: openapi_types.Date{Time: d.Date}}
		if d.Ayah != nil {
			item.Ayah = &DailyAyah{Ref: AyahRef{Surah: d.Ayah.Surah, Ayah: d.Ayah.Ayah}}
		}
		if d.Reminder != nil {
			r := toLocalized(d.Reminder)
			item.Reminder = &r
		}
		items = append(items, item)
	}
	return GetDaily200JSONResponse{Items: items}, nil
}
