package device

import "errors"

// ErrUnauthorized — нет такого токена (устройство удалено или токен неверный).
var ErrUnauthorized = errors.New("unauthorized")

// InvalidError — неверные данные запроса (→ 400).
type InvalidError struct{ Msg string }

func (e *InvalidError) Error() string { return e.Msg }

// Device — анонимное устройство. Ничего, что указывает на человека.
type Device struct {
	ID         string
	Platform   string // "ios"
	AppVersion string
	Locale     string // "" — не передан
	TimeZoneID string
	Country    string
}

// Push — куда и о чём слать уведомления.
type Push struct {
	APNsToken   string
	Environment string   // "production" | "sandbox"
	Topics      []string // "ramadan-TJ", "eid-TJ", "announcements"
}
