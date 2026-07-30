package server

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) GetRouter() *gin.Engine {

	r := gin.Default()
	r.Handle("GET", "/ws", s.Run)
	return r
}

func (s *Server) Run(c *gin.Context) {

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	for {

		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
		}

		log.Printf("received: %s", message)

		if err = ws.WriteMessage(messageType, message); err != nil {
			log.Println(err)
		}

	}

}
