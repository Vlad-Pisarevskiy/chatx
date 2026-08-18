package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	var choose int
	for {

		fmt.Println("Выберите действие: \n1 - Зарегистрироваться\n2 - Войти\n3 - Подключиться к чату")
		_, _ = fmt.Scan(&choose)
		switch choose {
		case 1:
			//var name string
			//var login string
			//var password string

			//fmt.Print("Введите имя: ")
			//fmt.Scan(&name)
			//
			//fmt.Print("Введите логин: ")
			//fmt.Scan(&login)
			//
			//fmt.Print("Введите пароль: ")
			//fmt.Scan(&password)

			var register = struct {
				Name     string `json:"name"`
				Login    string `json:"login"`
				Password string `json:"password"`
			}{
				Name:     "name",
				Login:    "login",
				Password: "my_password",
			}

			data, err := json.Marshal(register)
			if err != nil {
				log.Println(err)
			}
			rdr := bytes.NewReader(data)

			r, err := http.Post(fmt.Sprintf("http://%s/auth/register", addr), "application/json", rdr)
			if r.StatusCode != http.StatusOK {
				log.Println(r.StatusCode)
				return
			}
			fmt.Println(r.Body)
			if err != nil {
				log.Println(err)
				return
			}

		case 2:

			var login = struct {
				Login    string `json:"login"`
				Password string `json:"password"`
			}{
				Login:    "login",
				Password: "my_password",
			}

			data, _ := json.Marshal(login)

			loginData := bytes.NewReader(data)

			r, err := http.Post(fmt.Sprintf("http://%s/auth/login", addr), "application/json", loginData)
			if r.StatusCode != http.StatusOK {
				log.Println(r.StatusCode)
				return
			}
			fmt.Println(r.Body)
			if err != nil {
				log.Println(err)
				return
			}

		case 3:
		}
	}

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
