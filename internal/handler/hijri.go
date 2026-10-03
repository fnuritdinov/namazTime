package handler

import (
	"context"
	"errors"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"nTime/internal/hijrimonth"
)

// GET /v1/ramadan/{hijriYear}?country=TJ
func (s *Server) GetRamadan(ctx context.Context, req GetRamadanRequestObject) (GetRamadanResponseObject, error) {
	r, err := s.hijri.Ramadan(ctx, req.Params.Country, req.HijriYear)

	var ie *hijrimonth.InvalidError
	switch {
	case errors.As(err, &ie):
		return GetRamadan400JSONResponse{BadRequestJSONResponse(
			newError(ctx, "invalid_argument", ie.Msg))}, nil
	case errors.Is(err, hijrimonth.ErrNotFound):
		return GetRamadan404JSONResponse{NotFoundJSONResponse(
			newError(ctx, "not_found", fmt.Sprintf("no Ramadan %d data for %s yet", req.HijriYear, req.Params.Country)))}, nil
	case err != nil:
		return nil, err
	}

	duas := make([]Dua, 0, len(r.Duas))
	for _, d := range r.Duas {
		dua := Dua{Id: d.ID, Arabic: d.Arabic, Translations: toLocalized(d.Translations)}
		if d.Transliteration != "" {
			t := d.Transliteration
			dua.Transliteration = &t
		}
		if d.Source != "" {
			src := d.Source
			dua.Source = &src
		}
		duas = append(duas, dua)
	}

	laylat := openapi_types.Date{Time: r.LaylatAlQadr}
	resp := Ramadan{
		HijriYear:            r.HijriYear,
		Country:              r.Country,
		Status:               ConfirmationStatus(r.Status),
		StartDate:            openapi_types.Date{Time: r.Start},
		EndDate:              openapi_types.Date{Time: r.End},
		Length:               r.Length,
		EidAlFitr:            openapi_types.Date{Time: r.EidAlFitr},
		LaylatAlQadrExpected: &laylat,
		ConfirmedAt:          r.ConfirmedAt,
		Duas:                 duas,
	}
	if len(r.SourceName) > 0 {
		name := toLocalized(r.SourceName)
		resp.Source = &name
	}
	return GetRamadan200JSONResponse(resp), nil
}
