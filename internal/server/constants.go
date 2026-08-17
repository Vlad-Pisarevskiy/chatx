package server

import "time"

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
