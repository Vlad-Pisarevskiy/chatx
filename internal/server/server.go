package server

import (
	client2 "chatflow/internal/client"
	"chatflow/internal/service"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
	upgrader websocket.Upgrader
	service  *service.Service
}

func NewServer() *Server {

	return &Server{upgrader: websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}}
}

func (s *Server) GetRouter() *gin.Engine {

	r := gin.Default()

	auth := r.Group("/auth")
	{
		auth.POST("/register", s.Register)
		auth.POST("/login", s.Login)
	}

	r.GET("/ws", s.Run).Use()

	return r
}

func (s *Server) Register(c *gin.Context) {

	var client client2.Client
	if err := c.ShouldBind(&client); err != nil {
		c.JSON(401, gin.H{
			"error": "invalid data",
		})
	}

	if err := s.service.RegisterUser(c.Request.Context(), client); err != nil {
		c.JSON(401, gin.H{
			"error": err,
		})
	}
}

func (s *Server) Login(c *gin.Context) {

}

func (s *Server) Run(c *gin.Context) {

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	s.handle(conn)
}

func (s *Server) handle(conn *websocket.Conn) {

	msgChan := make(chan []byte, 256)
	defer close(msgChan)

	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(time.Second * 20)) })
	if err := conn.SetReadDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Println(err)
		return
	}
	conn.SetReadLimit(1024)

	go s.writer(conn, msgChan)

	s.reader(conn, msgChan)
}

func (s *Server) reader(conn *websocket.Conn, msgChan chan []byte) {

	defer conn.Close()

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

func (s *Server) writer(conn *websocket.Conn, msgChan chan []byte) {

	ticker := time.NewTicker(time.Second * 25)
	defer ticker.Stop()
	defer conn.Close()

	for {
		select {
		case msg, ok := <-msgChan:

			if !ok {
				return
			}

			if err := conn.SetWriteDeadline(time.Now().Add(time.Second * 30)); err != nil {
				log.Println(err)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
				return
			}

		case <-ticker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(time.Second * 30)); err != nil {
				log.Println(err)
				return
			}

			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
