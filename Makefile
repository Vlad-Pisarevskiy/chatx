server:
	go run cmd/chatx/main.go

integration:
	go test ./internal/test/

client:
	go run cmd/client/client.go

.PHONY: server client