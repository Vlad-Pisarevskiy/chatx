package protocol

type SentMessage struct {
	To      int    `json:"to"`
	Message string `json:"message"`
}
