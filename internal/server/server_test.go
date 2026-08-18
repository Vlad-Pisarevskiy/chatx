package server

import (
	"chatflow/internal/protocol"
	"chatflow/internal/repository/memory"
	"encoding/json"
	"net/http"
	"strings"

	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestServer_Websocket(t *testing.T) {

	repository := memory.NewRepository()
	service := NewMockService(repository)
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

	message := protocol.SendMessage{
		To:      "user2",
		Message: "Hello",
	}

	data, err := json.Marshal(message)
	assert.Nil(t, err)
	err = conn1.WriteMessage(websocket.TextMessage, data)
	assert.Nil(t, err)

	_, msg, err := conn2.ReadMessage()
	assert.Nil(t, err)
	assert.Equal(t, "Hello", string(msg))
}
