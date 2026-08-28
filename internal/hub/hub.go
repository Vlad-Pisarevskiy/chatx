package hub

import (
	"chatflow/internal/server"
	"sync"
)

const msgBufferSize = 256

// TODO: Hub занимается соединениями пользователей, добавление, удаление, доставка сообщенний
type Hub struct {
	conns map[int]map[*server.Conn]struct{}
	mu    sync.RWMutex
}

func NewHub() *Hub {

	return &Hub{
		conns: make(map[int]map[*server.Conn]struct{}, msgBufferSize),
		mu:    sync.RWMutex{},
	}
}
