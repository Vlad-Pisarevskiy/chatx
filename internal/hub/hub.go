package hub

import (
	"chatflow/internal/protocol"
	"sync"

	"github.com/gorilla/websocket"
)

const msgBufferSize = 256

type Hub struct {
	conns map[int]map[*Conn]struct{}
	mu    sync.RWMutex
}

func New() *Hub {

	return &Hub{
		conns: make(map[int]map[*Conn]struct{}, msgBufferSize),
		mu:    sync.RWMutex{},
	}
}

type Conn struct {
	ws     *websocket.Conn
	ch     chan protocol.SentMessage
	userID int
}

func (h *Hub) Add(userID int, conn *websocket.Conn, msgChan chan protocol.SentMessage) *Conn {

	userConn := &Conn{
		ws:     conn,
		ch:     msgChan,
		userID: userID,
	}

	h.mu.Lock()
	h.conns[userID] = map[*Conn]struct{}{ //TODO: идет затирание
		userConn: struct{}{},
	}
	h.mu.Unlock()

	return userConn
}

func (h *Hub) Send(message protocol.SentMessage) {

	h.mu.Lock()
	if _, ok := h.conns[message.To]; ok {
		for k, _ := range h.conns[message.To] {
			k.ch <- message
		}
	}

	h.mu.Unlock()
}

func (h *Hub) RemoveConn(c *Conn) {

	h.mu.Lock()
	delete(h.conns[c.userID], c)
	h.mu.Unlock()
}
