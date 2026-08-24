package model

import (
	"time"
)

type User struct {
	ID       int
	Name     string
	Login    string
	Password string
}

type Message struct {
	From      int
	Data      string
	CreatedAt time.Time
}

type UserFromDB struct {
	ID    int
	Name  string
	Login string
}
