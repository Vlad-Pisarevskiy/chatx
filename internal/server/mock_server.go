package server

import (
	"chatflow/internal/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MockServer struct {
	upgrader websocket.Upgrader
	service  Service
	conns    map[string][]*websocket.Conn
	mu       sync.RWMutex
}

func NewMockServer(service Service) *MockServer {

	return &MockServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBuffer,
			WriteBufferSize: writeBuffer,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		service: service,
		conns:   make(map[string][]*websocket.Conn),
		mu:      sync.RWMutex{},
	}
}

func (s *MockServer) GetMockRouter() *gin.Engine {

	r := gin.Default()

	auth := r.Group("/auth")
	{
		auth.POST("/register", s.Register)
		auth.POST("/login", s.Login)
	}

	r.GET("/ws", s.Run)

	return r
}

func (s *MockServer) Register(c *gin.Context) {

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

	c.JSON(http.StatusOK, gin.H{
		"success": "registred",
	})
}

func (s *MockServer) Login(c *gin.Context) {

	var request LoginRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid login data",
		})
		return
	}

	user, _, err := s.service.LoginUser(c.Request.Context(),
		request.Login,
		request.Password,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error:": err,
		})
		return
	}

	c.Set("login", user.Login)

	c.JSON(http.StatusOK, gin.H{
		"successful login": fmt.Sprintf("welcome, %s", user.Login),
	})

}

func (s *MockServer) Run(c *gin.Context) {

	userLogin, ok := c.Get("login")
	if !ok {
		log.Println("unexpected error")
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Добавляем соединение в пул
	login := fmt.Sprint(userLogin)

	s.mu.Lock()
	s.conns[login] = append(s.conns[login], conn)
	s.mu.Unlock()

	defer s.removeConn(login, conn)

	s.handle(conn)
}

func (s *MockServer) removeConn(userID string, conn *websocket.Conn) {

	conns := make([]*websocket.Conn, 0, len(s.conns[userID]))
	copy(conns, s.conns[userID])

	for i, c := range conns {
		if c == conn {
			conns = append(conns[:i], conns[i+1:]...)
		}
	}

	s.conns[userID] = conns
}

func (s *MockServer) handle(conn *websocket.Conn) {

	msgChan := make(chan protocol.SendMessage, msgBufferSize) // Канал для связи между горутинами
	done := make(chan struct{})

	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(readDeadline)) })
	if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		log.Println(err)
		return
	}

	conn.SetReadLimit(readLimit)

	go s.writer(conn, msgChan, done)

	s.reader(conn, msgChan, done)
}

// TODO: reader должен прочитать сообщение, через пакет протокола распарсить его, отправить данные в writer функцию
func (s *MockServer) reader(conn *websocket.Conn, msgChan chan protocol.SendMessage, done chan struct{}) {

	defer conn.Close()
	defer close(msgChan)
	defer close(done)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			done <- struct{}{}
			return
		}

		var sendMessage protocol.SendMessage
		if err = json.Unmarshal(message, &sendMessage); err != nil {
			log.Println(err)
			return
		}

		if err = conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
			log.Println(err)
			done <- struct{}{}
			return
		}
	}
}

// TODO: writer должен прочитать сообщение, понять кому оно адресовано, проитерироваться по всем соединениям получателя и зааписать в них сообщение
func (s *MockServer) writer(conn *websocket.Conn, msgChan chan protocol.SendMessage, done chan struct{}) {

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

			if err := s.sendMessage(msg.To, msg.Message); err != nil {
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

		case <-done:
			return
		}
	}
}

func (s *MockServer) sendMessage(login, message string) error {

	for _, c := range s.conns[login] {
		if err := c.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			return err
		}
	}
	return nil
}
