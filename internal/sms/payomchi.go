// Package sms отправляет SMS через сервис Payomchi (api.payomchi.tj).
package sms

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const payomchiBaseURL = "https://api.payomchi.tj/api/v1"

// ErrUnavailable — сервис SMS не ответил или ответил ошибкой.
var ErrUnavailable = errors.New("sms provider unavailable")

// Client — клиент Payomchi.
type Client struct {
	http     *http.Client
	base     string
	apiKey   string
	senderID string // "" — имя отправителя по умолчанию у Payomchi
}

// NewClientFromEnv читает SMS_API_KEY (обязательно) и SMS_SENDER_ID (необязательно).
func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("SMS_API_KEY")
	if key == "" {
		return nil, errors.New("SMS_API_KEY is required")
	}
	return &Client{
		http:     &http.Client{Timeout: 15 * time.Second},
		base:     payomchiBaseURL,
		apiKey:   key,
		senderID: os.Getenv("SMS_SENDER_ID"),
	}, nil
}

// Result — ответ Payomchi на отправку.
type Result struct {
	ID     flexID `json:"id"`
	Status string `json:"status"` // pending, ...
}

// flexID принимает id и строкой, и числом: в документации Payomchi это строка, на деле приходит число.
type flexID string

func (f *flexID) UnmarshalJSON(b []byte) error {
	*f = flexID(strings.Trim(string(b), `"`))
	return nil
}

// Send отправляет одно SMS. phone — только цифры с кодом страны (992901234567).
// idemKey — уникальный ключ отправки: при повторе с тем же ключом Payomchi не отправит второй раз.
func (c *Client) Send(ctx context.Context, phone, message, idemKey string) (Result, error) {
	body := map[string]string{"phone": phone, "message": message, "channel": "sms"}
	if c.senderID != "" {
		body["senderId"] = c.senderID
	}
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/user/sms", bytes.NewReader(raw))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idemKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// В тексте ошибки — только статус и начало ответа сервиса (без номера и ключа)
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return Result{}, fmt.Errorf("%w: status %d: %s", ErrUnavailable, resp.StatusCode, msg)
	}
	var r Result
	if err := json.Unmarshal(respBody, &r); err != nil {
		return Result{}, fmt.Errorf("%w: bad json: %v", ErrUnavailable, err)
	}
	return r, nil
}

// NewIdempotencyKey — случайный ключ для одной отправки.
func NewIdempotencyKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "zikr-" + hex.EncodeToString(b)
}
