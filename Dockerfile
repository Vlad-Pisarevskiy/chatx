FROM golang:1.26 AS base

WORKDIR /app

COPY go.sum ./
COPY go.mod ./
COPY config ./config

COPY web ./web
COPY internal ./internal
COPY cmd/chatx/main.go ./cmd/chatx/

RUN CGO_ENABLED=0 go build -o chat ./cmd/chatx

FROM alpine:latest
WORKDIR /app

COPY .env ./.env
COPY --from=base /app/chatx ./chat

ENTRYPOINT ["./chat"]