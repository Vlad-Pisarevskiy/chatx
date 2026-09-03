package protocol

import (
	"encoding/json"
	"time"
)

type Data struct {
	MessageType string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
}

type Send struct {
	ChatID      int    `json:"chat_id"`
	Body        string `json:"body"`
	ClientMsgID string `json:"client_msg_id"`
}

type Message struct {
	MessageID int       `json:"message_id"`
	Sender    string    `json:"sender"`
	Time      time.Time `json:"time"`
}

type Ack struct {
	ClientMsgID int `json:"client_msg_id"`
	MessageID   int `json:"message_id"`
}

type Error struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}
