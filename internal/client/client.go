package client

import (
	"bufio"
	"io"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) StartChat() {

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8082/ws", nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	c.handle(conn)
}

func (c *Client) handle(conn *websocket.Conn) {

	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	inputChan := make(chan string)
	go c.clientInput(inputChan)

	for {

		select {
		case input := <-inputChan:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(input)); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func (c *Client) clientInput(inputChan chan string) {

	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		if err == io.EOF {
			continue
		}
		inputChan <- msg
	}
}
