package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrDeviceLimit = errors.New("sms limit for this device is reached for today")
	ErrPhoneLimit  = errors.New("sms limit for this phone number is reached for today")
	ErrGlobalLimit = errors.New("sms service daily limit is reached")
)

// Sender — тот, кто реально отправляет SMS (Payomchi Client). В тестах — заглушка.
type Sender interface {
	Send(ctx context.Context, phone, message, idemKey string) (Result, error)
}

// Storage — журнал отправленных SMS (Postgres Store). В тестах — память.
type Storage interface {
	CountByDevice(ctx context.Context, deviceID string, since time.Time) (int, error)
	CountByPhone(ctx context.Context, phoneHash string, since time.Time) (int, error)
	CountAll(ctx context.Context, since time.Time) (int, error)
	Log(ctx context.Context, deviceID, phoneHash, providerID string) error
}

// Limits — лимиты на одни сутки.
type Limits struct {
	PerDevice int // с одного устройства
	PerPhone  int // на один номер
	PerDay    int // всего по сервису (защита баланса)
}

type Service struct {
	snd     Sender
	st      Storage
	limits  Limits
	hashKey []byte
	loc     *time.Location // «сутки» считаем по этому поясу
	log     *slog.Logger
	mu      sync.Mutex // проверка лимита и отправка — по одной, чтобы два запроса разом не обошли лимит
}

func NewService(snd Sender, st Storage, limits Limits, hashKey string, loc *time.Location, log *slog.Logger) *Service {
	return &Service{snd: snd, st: st, limits: limits, hashKey: []byte(hashKey), loc: loc, log: log}
}

// HashPhone — номер не храним, только HMAC-хеш (по нему считаем лимит на номер).
func (s *Service) HashPhone(phone string) string {
	m := hmac.New(sha256.New, s.hashKey)
	m.Write([]byte(phone))
	return hex.EncodeToString(m.Sum(nil))
}

// Send проверяет лимиты, отправляет SMS и возвращает, сколько ещё можно отправить сегодня с устройства.
func (s *Service) Send(ctx context.Context, deviceID, rawPhone, message string) (int, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return 0, err
	}
	hash := s.HashPhone(phone)

	now := time.Now().In(s.loc)
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)

	s.mu.Lock()
	defer s.mu.Unlock()

	byDevice, err := s.st.CountByDevice(ctx, deviceID, since)
	if err != nil {
		return 0, fmt.Errorf("count by device: %w", err)
	}
	if byDevice >= s.limits.PerDevice {
		return 0, ErrDeviceLimit
	}
	byPhone, err := s.st.CountByPhone(ctx, hash, since)
	if err != nil {
		return 0, fmt.Errorf("count by phone: %w", err)
	}
	if byPhone >= s.limits.PerPhone {
		return s.limits.PerDevice - byDevice, ErrPhoneLimit
	}
	all, err := s.st.CountAll(ctx, since)
	if err != nil {
		return 0, fmt.Errorf("count all: %w", err)
	}
	if all >= s.limits.PerDay {
		s.log.Warn("sms daily limit reached", "limit", s.limits.PerDay)
		return s.limits.PerDevice - byDevice, ErrGlobalLimit
	}

	res, err := s.snd.Send(ctx, phone, message, NewIdempotencyKey())
	if err != nil {
		s.log.Error("sms send failed", "device_id", deviceID, "err", err) // номер в лог не пишем
		return s.limits.PerDevice - byDevice, ErrUnavailable
	}
	if err := s.st.Log(ctx, deviceID, hash, string(res.ID)); err != nil {
		// SMS уже ушло — пользователю ошибку не возвращаем, но пишем в лог
		s.log.Error("sms log failed", "device_id", deviceID, "provider_id", string(res.ID), "err", err)
	}
	s.log.Info("sms sent", "device_id", deviceID, "provider_id", string(res.ID), "status", res.Status)
	return s.limits.PerDevice - byDevice - 1, nil
}
