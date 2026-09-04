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
	msgChan chan protocol.Send
	done    chan struct{}
	hub     *hub.Hub
	service *service.Service
}

func newSession(userID int, conn *websocket.Conn, hub2 *hub.Hub, service2 *service.Service) *session {

	return &session{
		userID:  userID,
		conn:    conn,
		msgChan: make(chan protocol.Send, msgBufferSize),
		done:    make(chan struct{}),
		hub:     hub2,
		service: service2,
	}
}

func (s *session) handle() {

	userConn := s.hub.Add(s.userID, s.conn, s.msgChan, s.done)

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
	if err != nil { // если при чтении сообщения происходит ошибка мы возвращаемся из функции записывая структуру в done канал, покаа все логино
		return err
	}

	var sendMessage protocol.Send
	var data protocol.Data
	if err = json.Unmarshal(message, &data); err != nil {
		log.Println(err)
		return nil
	}

	if data.MessageType == sendType {
		if err = json.Unmarshal(data.Payload, &sendMessage); err != nil {
			log.Println(err)
			return nil
		}

		if sendMessage.ChatID == nullID && sendMessage.PeerID == nullID {
			return err
		}

		if sendMessage.ChatID != nullID && sendMessage.PeerID != nullID {

		}

		if sendMessage.PeerID != nullID {
			chatID, err := s.service.GetOrCreateChat(context.Background(), sendMessage.PeerID, s.userID)
			if err != nil {
				return err
			}
			sendMessage.ChatID = chatID
		}

		if sendMessage.ChatID != nullID {
			s.sendToChat(sendMessage)
		}
	}

	if err = s.conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		return err
	}

	return nil
}

func (s *session) sendToChat(message protocol.Send) {

	if err := s.service.SendMessage(context.Background(), message, s.userID); err != nil {
		log.Println(err)
		return
	}

	s.hub.Send(message)
}

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

func (s *session) sendMessage(msg protocol.Send) {

	if err := s.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		log.Println(err)
		return
	}

	if err := s.conn.WriteMessage(websocket.TextMessage, []byte(msg.Body)); err != nil {
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
