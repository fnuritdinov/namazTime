.PHONY: up down logs run gen

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f app

run:
	docker compose up -d postgres redis
	set -a && . ./.env && set +a && go run ./cmd/server

gen:
	go tool oapi-codegen -config oapi-codegen.yaml api/openapi.yaml