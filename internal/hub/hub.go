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

const nullLength = 0

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

	if h.conns[userID] == nil {
		h.conns[userID] = map[*Conn]struct{}{}
	}
	h.conns[userID][userConn] = struct{}{}

	h.mu.Unlock()

	return userConn
}

func (h *Hub) Send(message protocol.SentMessage) {

	h.mu.RLock()
	chans := make([]chan protocol.SentMessage, nullLength, len(h.conns[message.To]))

	for k := range h.conns[message.To] {
		chans = append(chans, k.ch)
	}

	h.mu.RUnlock()

	for _, v := range chans {
		v <- message
	}
}

func (h *Hub) RemoveConn(c *Conn) {

	h.mu.Lock()

	delete(h.conns[c.userID], c)
	if len(h.conns[c.userID]) == nullLength {
		delete(h.conns, c.userID)
	}

	h.mu.Unlock()
}

func (h *Hub) Close() {

	h.mu.Lock()
	conns := make([]*Conn, nullLength, len(h.conns))

	for _, k := range h.conns {
		conn := slices.Collect(maps.Keys(k))
		conns = append(conns, conn...)
	}

	clear(h.conns)
	h.mu.Unlock()

	deadline := time.Now().Add(time.Second)
	for _, c := range conns {
		if err := c.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutdown"), deadline); err != nil {
			log.Println(err)
		}
		c.ws.Close()
	}
}
