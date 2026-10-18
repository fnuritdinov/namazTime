package push

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
)

type Config struct {
	KeyFile string // путь к AuthKey_XXXX.p8
	KeyID   string // 10 символов
	TeamID  string // 10 символов
	Topic   string // Bundle ID приложения, tj.namoz.app
}

// ConfigFromEnv читает APNS_KEY_FILE, APNS_KEY_ID, APNS_TEAM_ID, APNS_TOPIC.
func ConfigFromEnv() (Config, error) {
	c := Config{
		KeyFile: os.Getenv("APNS_KEY_FILE"),
		KeyID:   os.Getenv("APNS_KEY_ID"),
		TeamID:  os.Getenv("APNS_TEAM_ID"),
		Topic:   os.Getenv("APNS_TOPIC"),
	}
	if c.KeyFile == "" || c.KeyID == "" || c.TeamID == "" || c.Topic == "" {
		return Config{}, errors.New("APNS_KEY_FILE, APNS_KEY_ID, APNS_TEAM_ID and APNS_TOPIC are required")
	}
	return c, nil
}

type Message struct {
	Title string
	Body  string
	Topic string // подписка: ramadan-TJ, eid-TJ, announcements — уходит в пуш как поле "topic"
}

type Sender struct {
	sandbox    *apns2.Client
	production *apns2.Client
	topic      string
}

func NewSender(config Config) (*Sender, error) {
	key, err := token.AuthKeyFromFile(config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("read APNs key: %w", err)
	}
	tok := &token.Token{AuthKey: key, KeyID: config.KeyID, TeamID: config.TeamID}
	// Development()/Production() меняют сам клиент, поэтому клиентов два, а не один.
	return &Sender{
		sandbox:    apns2.NewTokenClient(tok).Development(),
		production: apns2.NewTokenClient(tok).Production(),
		topic:      config.Topic,
	}, nil
}

type Result struct {
	Status int
	Reason string
}

// Send отправляет один пуш. environment — "sandbox" или "production" из device_push.
func (s *Sender) Send(ctx context.Context, apnsToken, environment string, m Message) (Result, error) {
	client := s.production
	if environment == "sandbox" {
		client = s.sandbox
	}
	n := &apns2.Notification{
		DeviceToken: apnsToken,
		Topic:       s.topic,
		PushType:    apns2.PushTypeAlert,
		Priority:    apns2.PriorityHigh,
		Payload: payload.NewPayload().
			AlertTitle(m.Title).
			AlertBody(m.Body).
			Sound("default").
			Custom("topic", m.Topic),
	}
	res, err := client.PushWithContext(ctx, n)
	if err != nil {
		return Result{}, err // сеть, таймаут — Apple ответа не дала
	}
	return Result{Status: res.StatusCode, Reason: res.Reason}, nil
}
