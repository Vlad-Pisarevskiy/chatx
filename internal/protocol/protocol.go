package protocol

type SendMessage struct {
	To      string `json:"to"`
	Message string `json:"message"`
}
