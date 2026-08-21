package protocol

type SendMessage struct {
	From    int    `json:"from"`
	To      int    `json:"to"`
	Message string `json:"message"`
}
