package handler

import (
	"context"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"nTime/internal/hijri"
)

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

	year, start := hijri.NextRamadan(time.Now().UTC())

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
			Status:    ConfirmationStatus("expected"),
			HijriYear: year,
			StartDate: openapi_types.Date{Time: start},
		},
		SupportUrl: s.app.SupportURL,
	}, nil
}
