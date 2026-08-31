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

	userConn := s.hub.Add(s.userID, s.conn, s.msgChan)
	defer s.hub.RemoveConn(userConn)

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

	// TODO: Какая то фигня с каналами, какие то наадо закрывать каакие то нет, надо разобраться
	defer close(s.msgChan)
	defer close(s.done)

	for {
		if err := s.readMessage(); err != nil {
			log.Println(err)
			break
		}
	}
}

func (s *session) readMessage() error {

	_, message, err := s.conn.ReadMessage()
	if err != nil {
		s.done <- struct{}{}
		return err
	}

	var sentMessage protocol.SentMessage
	if err = json.Unmarshal(message, &sentMessage); err != nil {
		log.Println(err)
		return nil
	}

	s.sendToUser(sentMessage)

	if err = s.conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		s.done <- struct{}{}
		return err
	}

	return nil
}

func (s *session) sendToUser(message protocol.SentMessage) {

	if err := s.service.SendMessage(context.Background(), message, s.userID); err != nil {
		log.Println(err)
		return
	}

	s.hub.Send(message)

}

func (s *session) writer() {

	ticker := time.NewTicker(tickerTiming)

	defer ticker.Stop()
	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(s.conn)

	for {
		select {
		case msg, ok := <-s.msgChan:
			s.sendMessage(msg, ok)

		case <-ticker.C:
			s.sendTick()

		case <-s.done:
			return
		}
	}
}

func (s *session) sendMessage(msg protocol.SentMessage, ok bool) {

	if !ok {
		return
	}

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
