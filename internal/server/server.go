package server

import (
	"chatflow/internal/service"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	readBuffer    = 1024
	writeBuffer   = 1024
	msgBufferSize = 256
	readLimit     = 1024
	cookieMaxAge  = 1200

	readDeadline  = time.Second * 30
	writeDeadline = time.Second * 30
	tickerTiming  = time.Second * 25

	userIdKey = "userID"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Server struct {
	upgrader websocket.Upgrader
	service  *service.Service
}

func NewServer() *Server {

	return &Server{upgrader: websocket.Upgrader{
		ReadBufferSize:  readBuffer,
		WriteBufferSize: writeBuffer,
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

	r.GET("/ws", s.Run).Use(s.authorization())

	return r
}

func (s *Server) Register(c *gin.Context) {

	var registerRequest RegisterRequest
	if err := c.ShouldBind(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid data",
		})
		return
	}

	if err := s.service.RegisterUser(c.Request.Context(),
		registerRequest.Name,
		registerRequest.Login,
		registerRequest.Password,
	); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err,
		})
		return
	}
}

func (s *Server) Login(c *gin.Context) {

	var request LoginRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid login data",
		})
		return
	}

	_, token, err := s.service.LoginUser(c.Request.Context(),
		request.Login,
		request.Password,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error:": err,
		})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", token, cookieMaxAge, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

func (s *Server) authorization() gin.HandlerFunc {
	return func(c *gin.Context) {

		token, err := c.Cookie("token")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "missing token",
			})
			c.Abort()
		}

		id, err := s.service.CheckToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "authorization error",
			})
			c.Abort()
		}

		c.Set(userIdKey, id)
		c.Next()
	}
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

	msgChan := make(chan []byte, msgBufferSize)
	defer close(msgChan)

	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(readDeadline)) })
	if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		log.Println(err)
		return
	}

	conn.SetReadLimit(readLimit)

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

		if err = conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
			log.Println(err)
			return
		}
	}
}

func (s *Server) writer(conn *websocket.Conn, msgChan chan []byte) {

	ticker := time.NewTicker(tickerTiming)
	defer ticker.Stop()
	defer conn.Close()

	for {
		select {
		case msg, ok := <-msgChan:

			if !ok {
				return
			}

			if err := conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
				log.Println(err)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
				return
			}

		case <-ticker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
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
