package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
	upgrader websocket.Upgrader
}

func NewServer() *Server {

	return &Server{upgrader: websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: true,
	}}

}

func (s *Server) GetRouter() *gin.Engine {

	r := gin.Default()
	r.Handle("GET", "/ws", s.Run)
	return r
}

func (s *Server) Run(c *gin.Context) {

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	if err = conn.SetReadDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Println(err)
		return
	}

	handle(conn)
}

func handle(conn *websocket.Conn) {

	msgChan := make(chan []byte)
	defer close(msgChan)
	defer conn.Close()

	ticker := time.NewTicker(time.Second * 25)
	defer ticker.Stop()

	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(time.Second * 20)) })
	if err := conn.SetReadDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Println(err)
		return
	}

	go func() {

		defer conn.Close()

		for {
			select {
			case msg, ok := <-msgChan:

				if !ok {
					break
				}

				if err := conn.SetWriteDeadline(time.Now().Add(time.Second * 30)); err != nil {
					log.Println(err)
					break
				}

				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Println(err)
					break
				}

			case <-ticker.C:
				if err := conn.SetWriteDeadline(time.Now().Add(time.Second * 30)); err != nil {
					log.Println(err)
					break
				}

				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Println(err)
					break
				}
			}
		}
	}()

	for {

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("received: %s", message)

		msgChan <- message

		if err = conn.SetReadDeadline(time.Now().Add(time.Second * 30)); err != nil {
			log.Println(err)
			return
		}
	}
}
