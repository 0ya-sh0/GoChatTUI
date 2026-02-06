package client

import (
	"log"
	"net/url"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
)

func connect(username string, url url.URL) (*websocket.Conn, error) {
	log.Printf("connecting to %s", url.String())
	c, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return nil, err
	}
	message := protocol.ClaimUsernameRequest{
		Username: username,
	}
	err = c.WriteJSON(&message)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func listenWSEvents(conn *websocket.Conn, messages chan protocol.Message) {
	for {
		message := protocol.Message{}
		err := conn.ReadJSON(&message)
		if err != nil {
			close(messages)
			break
		}
		messages <- message
	}
}

func writeMessage(conn *websocket.Conn, message protocol.Message) error {
	return conn.WriteJSON(message)
}
