package protocol

import (
	"time"
)

const MESSAGE_TYPE_CHAT = "CHAT"
const MESSAGE_TYPE_BROADCAST = "BROADCAST"

type ClaimUsernameRequest struct {
	Username string `json:"username"`
}

type ForwardMessageRequest struct {
	ToUsername string `json:"toUsername"`
	Content    string `json:"content"`
}

type Message struct {
	Type         string    `json:"type"`
	FromUsername string    `json:"fromUsername"`
	ToUsername   string    `json:"toUsername"`
	Content      string    `json:"content"`
	Timestamp    time.Time `json:"timestamp"`
	Users        []string  `json:"users"`
}
