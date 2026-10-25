package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nTime/internal/sms"
)

// newSMSService включает SMS, только если заданы SMS_API_KEY и SMS_HASH_KEY. Иначе nil (эндпоинт отдаст 503).
func newSMSService(db *pgxpool.Pool, log *slog.Logger) *sms.Service {
	if os.Getenv("SMS_API_KEY") == "" || os.Getenv("SMS_HASH_KEY") == "" {
		log.Warn("sms disabled: SMS_API_KEY or SMS_HASH_KEY is empty")
		return nil
	}
	client, err := sms.NewClientFromEnv()
	if err != nil {
		log.Error("sms disabled", "err", err)
		return nil
	}
	perDay := 50
	if v, err := strconv.Atoi(os.Getenv("SMS_DAILY_LIMIT")); err == nil && v > 0 {
		perDay = v
	}
	loc, err := time.LoadLocation("Asia/Dushanbe")
	if err != nil {
		log.Error("sms disabled", "err", err)
		return nil
	}
	return sms.NewService(client, sms.NewStore(db),
		sms.Limits{PerDevice: 2, PerPhone: 2, PerDay: perDay},
		os.Getenv("SMS_HASH_KEY"), loc, log)
}
