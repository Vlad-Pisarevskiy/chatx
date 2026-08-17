package server

import (
	"chatflow/internal/protocol"
	"chatflow/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	conns    map[string][]*websocket.Conn // Держит все текущие соединения
	mu       sync.RWMutex
}

func NewServer() *Server {

	return &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBuffer,
			WriteBufferSize: writeBuffer,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		conns: make(map[string][]*websocket.Conn)}
}

func (s *Server) GetRouter() *gin.Engine {

	r := gin.Default()

	r.LoadHTMLFiles("web/login.html", "web/register.html")

	r.GET("/register", s.Registration)
	r.GET("/login", s.Authorization)

	auth := r.Group("/auth")
	{
		auth.POST("/register", s.Register)
		auth.POST("/login", s.Login)
	}

	ws := r.Group("/ws")
	//ws.Use(s.authorization())
	ws.GET("/", s.Run)

	return r
}

func (s *Server) Registration(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func (s *Server) Authorization(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
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

	user, token, err := s.service.LoginUser(c.Request.Context(),
		request.Login,
		request.Password,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error:": err,
		})
		return
	}

	// Устанавливает, кто может переходить на сайт по ссылкам
	c.SetSameSite(http.SameSiteLaxMode)

	// Задаем куки. Path определяет путь по которому будут работать куки. Secure определяет доступность для не https соединений
	c.SetCookie("token", token, cookieMaxAge, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"successful login": fmt.Sprintf("welcome, %s", user.Login),
	})
}

func (s *Server) Run(c *gin.Context) {

	userID, ok := c.Get(userIdKey)
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
	id := fmt.Sprint(userID)

	s.mu.Lock()
	s.conns[id] = append(s.conns[id], conn)
	s.mu.Unlock()

	defer s.removeConn(id, conn)

	s.handle(conn)
}

func (s *Server) removeConn(userID string, conn *websocket.Conn) {

	conns := make([]*websocket.Conn, 0, len(s.conns[userID]))
	copy(conns, s.conns[userID])

	for i, c := range conns {
		if c == conn {
			conns = append(conns[:i], conns[i+1:]...)
		}
	}

	s.conns[userID] = conns
}

func (s *Server) handle(conn *websocket.Conn) {

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
func (s *Server) reader(conn *websocket.Conn, msgChan chan protocol.SendMessage, done chan struct{}) {

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
func (s *Server) writer(conn *websocket.Conn, msgChan chan protocol.SendMessage, done chan struct{}) {

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

func (s *Server) sendMessage(id, message string) error {

	for _, c := range s.conns[id] {
		if err := c.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			return err
		}
	}
	return nil
}
