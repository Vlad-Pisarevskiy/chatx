package model

type User struct {
	ID       string
	Name     string
	Login    string
	Password string
}

type Message struct {
	From string
	To   string
	Data string
}
