package device

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

// Store — где хранятся устройства (в main — Repository, в тестах — заглушка).
type Store interface {
	Create(ctx context.Context, d Device, tokenHash string) error
	// Touch находит устройство по хэшу токена и обновляет last_seen_at; ErrUnauthorized, если нет.
	Touch(ctx context.Context, tokenHash string) (deviceID string, err error)
	SetPush(ctx context.Context, deviceID string, p Push) error
	Delete(ctx context.Context, deviceID string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

var (
	versionRe = regexp.MustCompile(`^\d{1,4}(\.\d{1,4}){0,3}$`)        // 1.0, 1.2.3
	localeRe  = regexp.MustCompile(`^[a-z]{2,3}([_-][A-Za-z]{2,4})?$`) // ru, ru_TJ, tg-TJ
	countryRe = regexp.MustCompile(`^[A-Z]{2}$`)
	apnsRe    = regexp.MustCompile(`^[0-9a-fA-F]{64,200}$`)
	topicRe   = regexp.MustCompile(`^((ramadan|eid)-[A-Z]{2}|announcements)$`)
)

const maxTopics = 20

// Register создаёт устройство и возвращает его id и токен.
// Токен показывается один раз: в базе хранится только SHA-256 от него (§16 ТЗ).
func (s *Service) Register(ctx context.Context, d Device) (id, token string, err error) {
	switch {
	case d.Platform != "ios":
		return "", "", &InvalidError{Msg: "platform must be ios"}
	case !versionRe.MatchString(d.AppVersion):
		return "", "", &InvalidError{Msg: "appVersion must look like 1.0.0"}
	case d.Locale != "" && !localeRe.MatchString(d.Locale):
		return "", "", &InvalidError{Msg: "locale must look like ru_TJ"}
	case d.Country != "" && !countryRe.MatchString(d.Country):
		return "", "", &InvalidError{Msg: "country must be ISO 3166-1 alpha-2, e.g. TJ"}
	}
	if d.TimeZoneID != "" {
		if _, err := time.LoadLocation(d.TimeZoneID); err != nil || d.TimeZoneID == "Local" {
			return "", "", &InvalidError{Msg: fmt.Sprintf("unknown timeZoneId %q", d.TimeZoneID)}
		}
	}

	d.ID = "dev_" + randomHex(12)
	token = randomToken()
	if err := s.store.Create(ctx, d, HashToken(token)); err != nil {
		return "", "", err
	}
	return d.ID, token, nil
}

// Authenticate — какое устройство владеет токеном.
func (s *Service) Authenticate(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrUnauthorized
	}
	return s.store.Touch(ctx, HashToken(token))
}

// SetPush сохраняет APNs-токен и подписки. Повторы тем убираются.
func (s *Service) SetPush(ctx context.Context, deviceID string, p Push) error {
	if !apnsRe.MatchString(p.APNsToken) {
		return &InvalidError{Msg: "apnsToken must be a hex string"}
	}
	if p.Environment != "production" && p.Environment != "sandbox" {
		return &InvalidError{Msg: "environment must be production or sandbox"}
	}
	if len(p.Topics) > maxTopics {
		return &InvalidError{Msg: fmt.Sprintf("at most %d topics", maxTopics)}
	}
	seen := map[string]bool{}
	topics := []string{}
	for _, t := range p.Topics {
		if !topicRe.MatchString(t) {
			return &InvalidError{Msg: fmt.Sprintf("unknown topic %q: use ramadan-XX, eid-XX or announcements", t)}
		}
		if !seen[t] {
			seen[t] = true
			topics = append(topics, t)
		}
	}
	p.Topics = topics
	return s.store.SetPush(ctx, deviceID, p)
}

// Delete удаляет устройство со всеми данными (право на удаление, §12.2 ТЗ).
func (s *Service) Delete(ctx context.Context, deviceID string) error {
	return s.store.Delete(ctx, deviceID)
}

// HashToken — SHA-256 токена в hex. Даже если базу украдут, токенами воспользоваться не смогут.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// randomToken — 32 случайных байта (256 бит): подобрать невозможно.
func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand: криптостойкий генератор, не math/rand
	return base64.RawURLEncoding.EncodeToString(b)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
