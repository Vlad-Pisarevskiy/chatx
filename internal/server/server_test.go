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

	conn3, r, err := dialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/ws/?login=user3", nil)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, r.StatusCode)

	message := protocol.SentMessage{
		To:      "user2",
		Message: "Hello From User 1",
	}
	data, err := json.Marshal(message)
	assert.Nil(t, err)

	message2 := protocol.SentMessage{
		To:      "user2",
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
