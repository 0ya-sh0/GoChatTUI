package server

import (
	"time"

	"github.com/gorilla/websocket"
)

const CHAT_TYPE = "CHAT"
const BROADCAST_TYPE = "BROADCAST"

type UserNameMessage struct {
	UserName string `json:"userName"`
}

type JoinUserRequest struct {
	UserName string
	conn     *websocket.Conn
}

type JoinedUser struct {
	UserName        string
	conn            *websocket.Conn
	sendMessageChan chan interface{}
}

type ChatMessageRequest struct {
	ToUserName string `json:"toUserName"`
	Content    string `json:"content"`
}

type MessageToClient struct {
	Type         string    `json:"type"`
	FromUserName string    `json:"fromUserName"`
	ToUserName   string    `json:"toUserName"`
	Content      string    `json:"content"`
	Timestamp    time.Time `json:"timestamp"`
	Users        []string  `json:"users"`
}
