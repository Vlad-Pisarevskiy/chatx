package client

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) StartChat() error {

	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Set("Origin", "client")

	conn, resp, err := dialer.Dial("localhost:8082/ws", header)
	if err != nil {
		return err
	}
	log.Println(resp.StatusCode)

	if err = conn.WriteMessage(websocket.TextMessage, []byte("Hello world")); err != nil {
		return err
	}
	return nil
}
