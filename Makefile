.PHONY: up down logs run

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f app

# запуск Go локально, базы — в Docker
run:
	docker compose up -d postgres redis
	set -a && . ./.env && set +a && go run ./cmd/server