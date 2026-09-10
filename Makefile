-include .env
export

GOOSE_DRIVER ?= postgres
GOOSE_DBSTRING ?= $(DATABASE_URL)
GOOSE_MIGRATION_DIR ?= migrations

chat:
	go run cmd/chat/main.go

integration:
	go test ./internal/test/

postgres:
	docker compose up -d postgres

client:
	go run cmd/client/client.go

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down -v

migrate-up:
	go tool goose up

migrate-down:
	go tool goose down

generate:
	go tool oapi-codegen --config api/codegen.yaml api/groups.yaml

.PHONY: server client compose integration migrate-up migrate-down migrate-status migrate-create generate
