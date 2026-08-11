package model

type Client struct {
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
