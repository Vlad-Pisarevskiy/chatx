package server

import (
	errors1 "chatflow/internal/app-errors"
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
	msgChan chan protocol.Data
	hub     *hub.Hub
	service *service.Service
	ctx     context.Context
	cancel  context.CancelFunc
}

func newSession(userID int, conn *websocket.Conn, hub2 *hub.Hub, service2 *service.Service) *session {

	var sess = &session{
		userID:  userID,
		conn:    conn,
		msgChan: make(chan protocol.Data, msgBufferSize),
		hub:     hub2,
		service: service2,
	}

	sess.ctx, sess.cancel = context.WithCancel(context.Background())

	return sess
}

func (s *session) handle() {

	userConn := s.hub.Add(s.userID, s.conn, s.msgChan, s.ctx.Done())

	defer s.hub.RemoveConn(userConn)
	defer s.cancel()
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

	if data.Type == sendType {
		if err = json.Unmarshal(data.Payload, &sendMessage); err != nil {
			log.Println(err)
			return nil
		}

		if sendMessage.ChatID == nullID && sendMessage.PeerID == nullID {
			return errors1.ErrIncorrectData
		}

		if sendMessage.ChatID != nullID && sendMessage.PeerID != nullID {
			return errors1.ErrIncorrectData
		}

		if sendMessage.ChatID != nullID {
			s.sendToChat(sendMessage)

		} else if sendMessage.PeerID != nullID {
			chatID, err := s.service.GetOrCreateChat(s.ctx, sendMessage.PeerID, s.userID)
			if err != nil {
				return err
			}
			sendMessage.ChatID = chatID
			s.sendToChat(sendMessage)
		}
	}

	if err = s.conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		return err
	}

	return nil
}

func (s *session) sendToChat(send protocol.Send) {

	message, userID, err := s.service.SendMessage(s.ctx, send, s.userID)
	if err != nil {
		log.Println(err)
		return
	}
	s.sendAck(send.ClientMsgID, message.Id, message.Time)

	msg, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
		return
	}

	data := protocol.Data{
		Type:    messageType,
		Payload: msg,
	}

	s.hub.Send(userID, &data)
}

func (s *session) sendAck(clientMsgID string, messageID int, time2 time.Time) {

	ack := protocol.Ack{
		ClientMsgID: clientMsgID,
		MessageID:   messageID,
		Time:        time2,
	}

	payload, err := json.Marshal(ack)
	if err != nil {
		log.Println(err)
		return
	}

	data := protocol.Data{
		Type:    ackType,
		Payload: payload,
	}

	select {
	case s.msgChan <- data:
	case <-s.ctx.Done():
		return

	}
}

// TODO: так, мне надо клиенту отправлять подтверждение что его сообщение пришло, как это сделать? С бд
// с бд надо возвращать структуру которая мне нужнаа для отправки и отпраавлять, парсинг происхходит на уровне фроонтаа
func (s *session) writer() {

	ticker := time.NewTicker(tickerTiming)

	defer ticker.Stop()

	for {
		select {
		case msg := <-s.msgChan:

			s.sendMessage(msg)

		case <-ticker.C:
			s.sendTick()

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *session) sendMessage(msg protocol.Data) {

	if err := s.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		log.Println(err)
		return
	}

	message, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}

	if err := s.conn.WriteMessage(websocket.TextMessage, message); err != nil {
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
