package device

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// memStore — устройства в памяти.
type memStore struct {
	byHash map[string]string // хэш токена → id
	push   map[string]Push
}

func newMem() *memStore { return &memStore{byHash: map[string]string{}, push: map[string]Push{}} }

func (m *memStore) Create(_ context.Context, d Device, hash string) error {
	m.byHash[hash] = d.ID
	return nil
}

func (m *memStore) Touch(_ context.Context, hash string) (string, error) {
	id, ok := m.byHash[hash]
	if !ok {
		return "", ErrUnauthorized
	}
	return id, nil
}

func (m *memStore) SetPush(_ context.Context, id string, p Push) error {
	m.push[id] = p
	return nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	for h, v := range m.byHash {
		if v == id {
			delete(m.byHash, h)
		}
	}
	return nil
}

const apns = "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"

func TestRegisterAndAuthenticate(t *testing.T) {
	store := newMem()
	svc := NewService(store)
	ctx := context.Background()

	id, token, err := svc.Register(ctx, Device{Platform: "ios", AppVersion: "1.0.0", Locale: "ru_TJ", Country: "TJ", TimeZoneID: "Asia/Dushanbe"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "dev_") || len(token) < 40 {
		t.Errorf("id=%s token=%s", id, token)
	}
	// В хранилище — хэш, а не сам токен
	if _, ok := store.byHash[token]; ok {
		t.Error("token stored in plain text")
	}

	got, err := svc.Authenticate(ctx, token)
	if err != nil || got != id {
		t.Errorf("authenticate: %s, %v", got, err)
	}
	if _, err := svc.Authenticate(ctx, "wrong"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("wrong token: %v", err)
	}

	// Два устройства — разные токены
	_, token2, _ := svc.Register(ctx, Device{Platform: "ios", AppVersion: "1.0"})
	if token2 == token {
		t.Error("tokens must be unique")
	}

	// После удаления токен не работает
	_ = svc.Delete(ctx, id)
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("deleted device still authenticates: %v", err)
	}
}

func TestRegister_Invalid(t *testing.T) {
	svc := NewService(newMem())
	var ie *InvalidError
	for name, d := range map[string]Device{
		"platform": {Platform: "android", AppVersion: "1.0.0"},
		"version":  {Platform: "ios", AppVersion: "latest"},
		"locale":   {Platform: "ios", AppVersion: "1.0.0", Locale: "русский"},
		"country":  {Platform: "ios", AppVersion: "1.0.0", Country: "tj"},
		"timezone": {Platform: "ios", AppVersion: "1.0.0", TimeZoneID: "Mars/Olympus"},
	} {
		if _, _, err := svc.Register(context.Background(), d); !errors.As(err, &ie) {
			t.Errorf("%s: want InvalidError, got %v", name, err)
		}
	}
}

func TestSetPush(t *testing.T) {
	store := newMem()
	svc := NewService(store)
	ctx := context.Background()

	err := svc.SetPush(ctx, "dev_1", Push{APNsToken: apns, Environment: "production",
		Topics: []string{"ramadan-TJ", "eid-TJ", "ramadan-TJ", "announcements"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.push["dev_1"].Topics; len(got) != 3 {
		t.Errorf("duplicates not removed: %v", got)
	}

	var ie *InvalidError
	bad := map[string]Push{
		"apns":        {APNsToken: "not-hex", Environment: "production"},
		"environment": {APNsToken: apns, Environment: "dev"},
		"topic":       {APNsToken: apns, Environment: "production", Topics: []string{"fajr-TJ"}},
		"lowercase":   {APNsToken: apns, Environment: "production", Topics: []string{"ramadan-tj"}},
	}
	for name, p := range bad {
		if err := svc.SetPush(ctx, "dev_1", p); !errors.As(err, &ie) {
			t.Errorf("%s: want InvalidError, got %v", name, err)
		}
	}
}
