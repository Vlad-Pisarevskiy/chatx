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
	From      int       `json:"from"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

type UserFromDB struct {
	ID    int
	Name  string
	Login string
}
