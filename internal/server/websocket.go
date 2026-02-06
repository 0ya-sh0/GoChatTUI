package server

import (
	"log"
	"net/http"
	"time"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{}

func (b *Broker) HandleWebsocketConnection(w http.ResponseWriter, r *http.Request) {
	log.Print("req connection")
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("err upgrade:", err)
		return
	}
	log.Print("req connection waiting for username req")
	go waitForUsernameClaim(conn, b.joinUserRequests)
}

func waitForUsernameClaim(conn *websocket.Conn, joinUserRequests chan<- JoinUserRequest) {
	log.Print("waiting for conn2")
	claimedUsername := make(chan *protocol.ClaimUsernameRequest)
	defer close(claimedUsername)

	go func() {
		var claimUsernameRequest protocol.ClaimUsernameRequest
		err := conn.ReadJSON(&claimUsernameRequest)
		if err == nil {
			claimedUsername <- &claimUsernameRequest
		} else {
			log.Print("err username:", err)
			claimedUsername <- nil
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		conn.Close()
		log.Print("closing client no username requested")
		return
	case username := <-claimedUsername:
		if username == nil {
			conn.Close()
			return
		}
		joinUserRequests <- JoinUserRequest{
			username: username.Username,
			conn:     conn,
		}
	}
}

func messageSender(conn *websocket.Conn, inbox <-chan interface{}) {
	draining := false
	for {
		message, ok := <-inbox
		if !ok {
			return
		}
		if draining {
			continue
		}
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		err := conn.WriteJSON(message)
		if err != nil {
			draining = true
			conn.Close()
		}
	}
}

func messageReciever(conn *websocket.Conn, outbox chan<- protocol.Message, username string, kickOutUser chan<- string) {
	for {
		var message protocol.ForwardMessageRequest
		err := conn.ReadJSON(&message)
		if err != nil {
			conn.Close()
			kickOutUser <- username
			break
		}
		log.Print(username, message)
		outbox <- protocol.Message{
			Type:         protocol.MESSAGE_TYPE_CHAT,
			FromUsername: username,
			ToUsername:   message.ToUsername,
			Content:      message.Content,
			Timestamp:    time.Now(),
		}
	}
}
