package server

import "time"

const (
	readBuffer    = 1024
	writeBuffer   = 1024
	msgBufferSize = 256
	readLimit     = 1024
	cookieMaxAge  = int((time.Hour * 24 * 30) / time.Second)

	readDeadline  = time.Second * 30
	writeDeadline = time.Second * 30
	tickerTiming  = time.Second * 25

	messageType = "message"
	userIdKey   = "userID"
	tokenKey    = "token"
	sendType    = "send"
	peerID      = "peer_id"
	ackType     = "ack"
	emptyPeer   = ""
	nullID      = 0
)
