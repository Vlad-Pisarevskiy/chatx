package server

import (
	"chatflow/internal/hub"
	"chatflow/internal/protocol"
	"chatflow/internal/service"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type session struct {
	userID  int
	conn    *websocket.Conn
	msgChan chan protocol.SentMessage
	done    chan struct{}
	hub     *hub.Hub
	service *service.Service
}

func newSession(userID int, conn *websocket.Conn, hub2 *hub.Hub, service2 *service.Service) *session {

	return &session{
		userID:  userID,
		conn:    conn,
		msgChan: make(chan protocol.SentMessage, msgBufferSize),
		done:    make(chan struct{}),
		hub:     hub2,
		service: service2,
	}
}

func (s *session) handle() {

	userConn := s.hub.Add(s.userID, s.conn, s.msgChan, s.done) // добавляем пользователя в хаб соединений
	defer s.hub.RemoveConn(userConn)
	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(s.conn)

	s.conn.SetPongHandler(func(string) error { return s.conn.SetReadDeadline(time.Now().Add(readDeadline)) })
	if err := s.conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		log.Println(err)
		return
	}

	s.conn.SetReadLimit(readLimit)

	go s.writer()

	s.reader()
}

func (s *session) reader() {

	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(s.conn)

	defer close(s.done) // также закрываем канал сигнализирующий о завершении работы программы

	for {
		if err := s.readMessage(); err != nil {
			log.Println(err)
			break // не получилось прочитать сообщение - выходим, после закрываются каналы
		}
	}
}

func (s *session) readMessage() error {

	_, message, err := s.conn.ReadMessage()
	if err != nil { // если при чтении сообщения происходит ошибка мы возвращаемся из функции записывая структуру в done канал, покаа все логино
		return err
	}

	var sentMessage protocol.SentMessage
	if err = json.Unmarshal(message, &sentMessage); err != nil {
		log.Println(err)
		return nil // если не получилось распарсить, то смысл из-за этого сбрасывать соединение
	}

	s.sendToUser(sentMessage)

	if err = s.conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		return err
	}

	return nil
}

func (s *session) sendToUser(message protocol.SentMessage) {

	if err := s.service.SendMessage(context.Background(), message, s.userID); err != nil {
		log.Println(err)
		return
	}
	// странно, если не получилось отправить сообщение то ничего не возвращаем
	// хотя это же просто запись в бд, если не записалось то смысл из за этого соединение обрываать
	s.hub.Send(message)
}

// Пишет присланные сообщения
func (s *session) writer() {

	ticker := time.NewTicker(tickerTiming)

	defer ticker.Stop()

	for {
		select {
		case msg := <-s.msgChan:
			s.sendMessage(msg)

		case <-ticker.C:
			s.sendTick()

		case <-s.done: // когда приходит структура возвращаемся
			return
		}
	}
}

func (s *session) sendMessage(msg protocol.SentMessage) {

	if err := s.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		log.Println(err)
		return
	}

	if err := s.conn.WriteMessage(websocket.TextMessage, []byte(msg.Message)); err != nil {
		log.Println(err)
		return
	}

}

func (s *session) sendTick() {

	if err := s.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		log.Println(err)
		return
	}

	if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		log.Println(err)
		return
	}
}
