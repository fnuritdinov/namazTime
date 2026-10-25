package sms

import (
	"errors"
	"strings"
)

// ErrBadPhone — номер не похож на таджикский мобильный.
var ErrBadPhone = errors.New("phone must be a Tajikistan number: 992 and 9 digits")

// NormalizePhone приводит номер к виду Payomchi: только цифры с кодом страны.
// Принимает "+992 90 123-45-67", "992901234567", "(992) 901234567".
// Пока разрешены только номера Таджикистана (992 + 9 цифр) — защита от дорогих зарубежных номеров.
func NormalizePhone(s string) (string, error) {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
			// пропускаем оформление
		default:
			return "", ErrBadPhone
		}
	}
	p := b.String()
	if len(p) != 12 || !strings.HasPrefix(p, "992") {
		return "", ErrBadPhone
	}
	return p, nil
}
