package handler

import (
	"context"

	"nTime/internal/prayertime"
)

// GET /v1/calculation-methods
func (s *Server) ListCalculationMethods(ctx context.Context, req ListCalculationMethodsRequestObject) (ListCalculationMethodsResponseObject, error) {
	items := make([]CalculationMethod, 0, len(prayertime.Methods))
	for _, m := range prayertime.Methods {
		items = append(items, CalculationMethod{
			Id:        CalculationMethodId(m.ID),
			Title:     toLocalized(m.Title),
			FajrAngle: m.FajrAngle,
			Isha: IshaRule{
				Type:  IshaRuleType(m.Isha.Type),
				Value: m.Isha.Value,
			},
		})
	}
	return ListCalculationMethods200JSONResponse{Items: items}, nil
}
