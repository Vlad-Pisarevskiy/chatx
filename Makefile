-include .env
export

GOOSE_DRIVER ?= postgres
GOOSE_DBSTRING ?= $(DATABASE_URL)
GOOSE_MIGRATION_DIR ?= migrations

chat:
	go run cmd/chatx/main.go

integration:
	go test ./internal/test/

client:
	go run cmd/client/client.go

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

migrate-up:
	go tool goose up

migrate-down:
	go tool goose down

.PHONY: server client compose integration migrate-up migrate-down migrate-status migrate-create
