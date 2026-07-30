package client

import (
	"bufio"
	"log"
	"os"

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

	for {

		inputChan := make(chan string)
		go c.clientInput(inputChan)

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
	msg, _ := reader.ReadString('\n')
	inputChan <- msg
}
