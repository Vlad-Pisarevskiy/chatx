package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	cr "github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) StartChat(addr string) {

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/ws", addr), nil)
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
	go reader(conn)

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

	r := bufio.NewReader(os.Stdin)
	for {
		msg, err := r.ReadString('\n')
		if err == io.EOF {
			continue
		}
		inputChan <- msg
	}
}

func reader(conn *websocket.Conn) {

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}
	cr.Blue(string(msg))

}
