package handler

import (
	"context"
	"log/slog"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// У /config нет параметра страны — Рамадан показываем для основной аудитории.
const defaultCountry = "TJ"

// AppInfo — настройки для /v1/config (заполняются в main из config.Config).
type AppInfo struct {
	MinSupportedVersion string
	LatestVersion       string
	SupportURL          string
	FeatureSync         bool
	FeatureQuranSearch  bool
	ContentVersions     map[string]int
}

// GET /v1/config
func (s *Server) GetConfig(ctx context.Context, req GetConfigRequestObject) (GetConfigResponseObject, error) {
	versions := s.app.ContentVersions
	if versions == nil {
		versions = map[string]int{} // в JSON — {}, а не null
	}

	// Даты из базы (Шуро), иначе — расчёт. /config не должен падать из-за базы:
	// при ошибке NextRamadan всё равно возвращает расчётную дату.
	r, err := s.hijri.NextRamadan(ctx, defaultCountry, time.Now().UTC())
	if err != nil {
		slog.Warn("config: next ramadan from db failed, using calculated", "err", err)
	}

	return GetConfig200JSONResponse{
		MinSupportedVersion: s.app.MinSupportedVersion,
		LatestVersion:       s.app.LatestVersion,
		Features: Features{
			Sync:               s.app.FeatureSync,
			OfficialTimetables: true,
			QuranSearch:        s.app.FeatureQuranSearch,
		},
		ContentVersions: versions,
		Ramadan: &RamadanSummary{
			Status:    ConfirmationStatus(r.Status),
			HijriYear: r.HijriYear,
			StartDate: openapi_types.Date{Time: r.Start},
		},
		SupportUrl: s.app.SupportURL,
	}, nil
}
