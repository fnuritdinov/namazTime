package main

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"nTime/internal/city"
	"nTime/internal/config"
	"nTime/internal/device"
	"nTime/internal/handler"
	"nTime/internal/hijrimonth"
	"nTime/internal/official"
	"nTime/internal/schedule"
)

// app — всё, что собрано из базы: API и то, что нужно main отдельно.
type app struct {
	api        *handler.Server
	devices    *device.Service    // для проверки токена в middleware
	deviceRepo *device.Repository // для фоновой очистки устройств
}

// newApp собирает репозитории → сервисы → API.
func newApp(db *pgxpool.Pool, cfg config.Config) app {
	cityRepo := city.NewRepository(db)
	officialRepo := official.NewRepository(db)
	hijriRepo := hijrimonth.NewRepository(db)
	deviceRepo := device.NewRepository(db)

	scheduleSvc := schedule.NewService(cityRepo, officialRepo, hijriRepo)
	hijriSvc := hijrimonth.NewService(hijriRepo)
	deviceSvc := device.NewService(deviceRepo)

	api := handler.NewServer(cityRepo, scheduleSvc, hijriSvc, deviceSvc, handler.AppInfo{
		MinSupportedVersion: cfg.MinAppVersion,
		LatestVersion:       cfg.LatestAppVersion,
		SupportURL:          cfg.SupportURL,
		FeatureSync:         cfg.FeatureSync,
		FeatureQuranSearch:  cfg.FeatureQuranSearch,
		ContentVersions:     cfg.ContentVersions,
	})

	return app{api: api, devices: deviceSvc, deviceRepo: deviceRepo}
}
