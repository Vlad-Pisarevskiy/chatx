package server

import (
	"chatflow/config"
	"chatflow/internal/protocol"
	"chatflow/internal/repository/postgres"
	service2 "chatflow/internal/service"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestServer_Websocket(t *testing.T) {

	pool, err := pgxpool.New(context.Background(), config.DBPath())
	assert.Nil(t, err)
	defer pool.Close()

	repository := postgres.NewRepository(pool)
	service := service2.NewService(repository)
	server := NewServer(service)

	router := server.GetRouter()

	srv := httptest.NewServer(router)

	dialer := websocket.DefaultDialer

	conn1, r, err := dialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/ws/?login=user1", nil)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, r.StatusCode)

	conn2, r, err := dialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/ws/?login=user2", nil)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, r.StatusCode)

	conn3, r, err := dialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/ws/?login=user3", nil)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, r.StatusCode)

	message := protocol.SentMessage{
		To:      1,
		Message: "Hello From User 1",
	}
	data, err := json.Marshal(message)
	assert.Nil(t, err)

	message2 := protocol.SentMessage{
		To:      2,
		Message: "Hello From User 3",
	}
	data2, err := json.Marshal(message2)
	assert.Nil(t, err)

	err = conn1.WriteMessage(websocket.TextMessage, data)
	assert.Nil(t, err)
	err = conn3.WriteMessage(websocket.TextMessage, data2)
	assert.Nil(t, err)

	_, msg, err := conn2.ReadMessage()
	assert.Nil(t, err)
	assert.Equal(t, "Hello From User 1", string(msg))

	_, msg, err = conn2.ReadMessage()
	assert.Nil(t, err)
	assert.Equal(t, "Hello From User 3", string(msg))
}
