package server

import (
	"chatflow/internal/hub"
	"chatflow/internal/protocol"
	"chatflow/internal/service"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

//type Service interface {
//	RegisterUser(ctx context.Context, name, login, password string) error
//	LoginUser(ctx context.Context, login, password string) (*model.User, string, error)
//	CheckToken(ctx context.Context, token string) (int, error)
//	GetUsers(ctx context.Context) ([]*model.UserFromDB, error)
//	FindUserByID(ctx context.Context, id int) (*model.UserFromDB, error)
//	GetUsersExcept(ctx context.Context, id int) ([]*model.UserFromDB, error)
//	ChatExists(ctx context.Context, from, to int) (bool, error)
//	StartChat(ctx context.Context, from, to int) (int, error)
//	SentMessage(ctx context.Context, chatID, from int, message string) error
//}

type Server struct {
	upgrader websocket.Upgrader
	service  *service.Service
	hub      *hub.Hub
}

type Conn struct {
	ws     *websocket.Conn
	ch     chan protocol.SentMessage
	userID int
}

func New(service *service.Service, hub *hub.Hub) *Server {

	return &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBuffer,
			WriteBufferSize: writeBuffer,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		service: service,
		hub:     hub}
}

func (s *Server) GetRouter() *gin.Engine {

	r := gin.Default()

	r.LoadHTMLFiles("web/login.html", "web/register.html", "web/users.html")

	r.GET("/register", s.Registration)
	r.GET("/login", s.Authorization)

	auth := r.Group("/auth")
	{
		auth.POST("/register", s.Register)
		auth.POST("/login", s.Login)
		auth.POST("/logout", s.Logout)
	}

	ws := r.Group("/ws")
	ws.Use(s.authorization())
	ws.GET("/", s.Run)

	u := r.Group("/users")
	u.Use(s.pageAuthorization())
	{
		u.GET("/", s.Chats)
		u.GET("/:userID/messages", s.LoadMessages)
	}

	return r
}

func (s *Server) Registration(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func (s *Server) Authorization(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (s *Server) Chats(c *gin.Context) {

	id, ok := c.Get(userIdKey)
	if !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "no such user",
		})
		return
	}

	users, err := s.service.GetUsersExcept(c.Request.Context(), id.(int))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	me, err := s.service.FindUserByID(c.Request.Context(), id.(int))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "users.html", gin.H{
		"Users": users,
		"Me":    me.Name,
		"MyID":  me.ID,
	})
}

func (s *Server) Logout(c *gin.Context) {

	token, err := c.Cookie("token")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	if err = s.service.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	c.Status(http.StatusOK)
}

// TODO: Загрузка сообщений происходит по айди юзера, скорее всего надо будет на айди чата поменять
func (s *Server) LoadMessages(c *gin.Context) {

	userFrom, ok := c.Get(userIdKey)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown user",
		})
		return
	}

	userTo := c.Param("userID")
	userID, err := strconv.Atoi(userTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown user id",
		})
		return
	}

	messages, err := s.service.LoadMessages(c.Request.Context(), userFrom.(int), userID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (s *Server) Register(c *gin.Context) {

	var registerRequest RegisterRequest
	if err := c.ShouldBind(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid data",
		})
		return
	}

	request := service.RegisterInput{
		Name:     registerRequest.Name,
		Login:    registerRequest.Login,
		Password: registerRequest.Password,
	}

	if err := s.service.RegisterUser(c.Request.Context(), request); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": "registered",
	})
}

func (s *Server) Login(c *gin.Context) {

	var request LoginRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid login data",
		})
		return
	}

	user, token, err := s.service.LoginUser(c.Request.Context(), //TODO: Токен никак не обрабатывается
		request.Login,
		request.Password,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error:": err.Error(),
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

	id := userID.(int)

	// TODO: Переделать и вернуть функцию
	//И отдельно: &Conn{...} создан прямо в литерале, указатель нигде не сохранён. Даже дописав removeConn, тебе нечего будет ему передать — ключ анонимный. Указатель надо положить в
	//  переменную и протащить до места удаления (и до reader/writer, которые сейчас получают conn, msgChan, done, userID четырьмя параметрами — вместо этого логичнее передавать один
	//  *Conn, ради него он и заводился).

	s.handle(id, conn)
}

func (s *Server) handle(userID int, conn *websocket.Conn) {

	msgChan := make(chan protocol.SentMessage, msgBufferSize) // Канал для связи между горутинами

	userConn := s.hub.Add(userID, conn, msgChan)
	defer s.hub.RemoveConn(userConn)

	done := make(chan struct{})

	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(readDeadline)) })
	if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		log.Println(err)
		return
	}

	conn.SetReadLimit(readLimit)

	go s.writer(conn, msgChan, done)

	s.reader(conn, msgChan, done, userID)
}

func (s *Server) reader(conn *websocket.Conn, msgChan chan protocol.SentMessage, done chan struct{}, userID int) {

	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(conn)

	// TODO: Какая то фигня с каналами, какие то наадо закрывать каакие то нет, надо разобраться
	defer close(msgChan)
	defer close(done)

	for {
		if err := s.readMessage(conn, userID, done); err != nil {
			log.Println(err)
			break
		}
	}
}

func (s *Server) readMessage(conn *websocket.Conn, userID int, done chan struct{}) error {

	_, message, err := conn.ReadMessage()
	if err != nil {
		done <- struct{}{}
		return err
	}

	var sentMessage protocol.SentMessage
	if err = json.Unmarshal(message, &sentMessage); err != nil {
		log.Println(err)
		return nil
	}

	s.sendToUser(sentMessage, userID)

	if err = conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		done <- struct{}{}
		return err
	}

	return nil
}

func (s *Server) sendToUser(message protocol.SentMessage, userID int) {

	if err := s.service.SendMessage(context.Background(), message, userID); err != nil {
		log.Println(err)
		return
	}

	s.hub.Send(message)

}

func (s *Server) writer(conn *websocket.Conn, msgChan chan protocol.SentMessage, done chan struct{}) {

	ticker := time.NewTicker(tickerTiming)

	defer ticker.Stop()
	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(conn)

	for {
		select {
		case msg, ok := <-msgChan:
			sendMessage(conn, msg, ok)

		case <-ticker.C:
			sendTick(conn)

		case <-done:
			return
		}
	}
}

func sendMessage(conn *websocket.Conn, msg protocol.SentMessage, ok bool) {

	if !ok {
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Message)); err != nil {
		log.Println(err)
		return
	}

	//TODO: неправильный порядок надо сравниить с тиком и выровнять
	if err := conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		log.Println(err)
		return
	}
}

func sendTick(conn *websocket.Conn) {

	if err := conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		log.Println(err)
		return
	}

	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		log.Println(err)
		return
	}
}

func (s *Server) authorization() gin.HandlerFunc {
	return func(c *gin.Context) {

		token, err := c.Cookie(tokenKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "missing token",
			})
			c.Abort()
			return
		}

		id, err := s.service.CheckToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "authorization error",
			})
			c.Abort()
			return
		}

		c.Set(userIdKey, id)
		c.Next()
	}
}

func (s *Server) pageAuthorization() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(tokenKey)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		id, err := s.service.CheckToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "authorization error",
			})
			c.Abort()
			return
		}

		c.Set(userIdKey, id)
		c.Next()
	}
}
