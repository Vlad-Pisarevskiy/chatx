-include .env
export

GOOSE_DRIVER ?= postgres
GOOSE_DBSTRING ?= $(DATABASE_URL)
GOOSE_MIGRATION_DIR ?= migrations

server:
	go run cmd/chatx/main.go

integration:
	go test ./internal/test/

client:
	go run cmd/client/client.go

compose:
	docker compose up -d

migrate-up:
	go tool goose up

migrate-down:
	go tool goose down

.PHONY: server client compose integration migrate-up migrate-down migrate-status migrate-create
