// sms-test — отправляет одно тестовое SMS через Payomchi.
//
//	go run ./cmd/sms-test -phone 992901234567 -message "Test Zikr"
//
// Нужен SMS_API_KEY (и по желанию SMS_SENDER_ID) в переменных окружения.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"nTime/internal/sms"
)

func main() {
	phone := flag.String("phone", "", "номер получателя: 992XXXXXXXXX")
	message := flag.String("message", "", "текст SMS")
	flag.Parse()

	p, err := sms.NormalizePhone(*phone)
	if err != nil || *message == "" {
		flag.Usage()
		log.Fatalf("-phone (992 + 9 цифр) и -message обязательны: %v", err)
	}
	c, err := sms.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r, err := c.Send(ctx, p, *message, sms.NewIdempotencyKey())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ok: id=%s status=%s\n", r.ID, r.Status)
}
