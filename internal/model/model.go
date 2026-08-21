package model

type User struct {
	ID       int
	Name     string
	Login    string
	Password string
}

type Message struct {
	From string
	To   string
	Data string
}

type UserFromDB struct {
	ID    int
	Name  string
	Login string
}
