// push-test — отправляет один тестовый пуш через APNs.
//
//	go run ./cmd/push-test -key AuthKey_ABC123DEFG.p8 -key-id ABC123DEFG -team-id TEAM123456 \
//	    -topic cloud.zikr.app -token <64 hex> -env sandbox
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
)

func main() {
	keyFile := flag.String("key", "", "путь к ключу .p8")
	keyID := flag.String("key-id", "", "Key ID (10 символов)")
	teamID := flag.String("team-id", "", "Team ID (10 символов)")
	topic := flag.String("topic", "", "Bundle ID приложения")
	deviceToken := flag.String("token", "", "токен устройства (hex)")
	env := flag.String("env", "sandbox", "sandbox (сборка из Xcode) или production (TestFlight/App Store)")
	title := flag.String("title", "Zikr", "заголовок")
	body := flag.String("body", "Тестовое уведомление", "текст")
	port := flag.String("port", "", "порт APNs: пусто = 443, можно 2197")
	flag.Parse()

	if *keyFile == "" || *keyID == "" || *teamID == "" || *topic == "" || *deviceToken == "" {
		flag.Usage()
		log.Fatal("все параметры обязательны")
	}

	// 1. Ключ .p8 → подпись запросов (JWT библиотека делает и обновляет сама)
	authKey, err := token.AuthKeyFromFile(*keyFile)
	if err != nil {
		log.Fatalf("read key: %v", err)
	}
	tok := &token.Token{AuthKey: authKey, KeyID: *keyID, TeamID: *teamID}

	// 2. Клиент: sandbox или production — это разные адреса Apple
	client := apns2.NewTokenClient(tok)
	switch *env {
	case "sandbox":
		client = client.Development()
	case "production":
		client = client.Production()
	default:
		log.Fatalf("-env: sandbox или production, получено %q", *env)
	}
	if *port != "" {
		client.Host = client.Host + ":" + *port
	}

	// 3. Само уведомление
	n := &apns2.Notification{
		DeviceToken: *deviceToken,
		Topic:       *topic,
		PushType:    apns2.PushTypeAlert,
		Priority:    apns2.PriorityHigh,
		Payload:     payload.NewPayload().AlertTitle(*title).AlertBody(*body).Sound("default"),
	}

	// 4. Отправка
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := client.PushWithContext(ctx, n)
	if err != nil {
		log.Fatalf("push: %v", err) // сеть, неверный ключ и т.п.
	}

	fmt.Printf("status=%d reason=%q apns-id=%s\n", res.StatusCode, res.Reason, res.ApnsID)
	if !res.Sent() {
		log.Fatal("Apple не приняла уведомление, см. reason")
	}
	fmt.Println("отправлено")
}
