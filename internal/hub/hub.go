package hub

import (
	"chatflow/internal/protocol"
	"log"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	conns map[int]map[*Conn]struct{}
	mu    sync.RWMutex
}

func New() *Hub {

	return &Hub{
		conns: make(map[int]map[*Conn]struct{}),
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

func (h *Hub) Close() {

	conns := make([]*Conn, 0, len(h.conns))

	h.mu.Lock()
	for _, k := range h.conns {
		conn := slices.Collect(maps.Keys(k))
		conns = append(conns, conn...)
	}

	h.conns = map[int]map[*Conn]struct{}{}

	h.mu.Unlock()

	for _, c := range conns {
		if err := c.ws.WriteControl(websocket.CloseMessage, []byte("server shutdown"), time.Now().Add(time.Second)); err != nil {
			log.Println(err)
		}
		c.ws.Close()
	}
}
