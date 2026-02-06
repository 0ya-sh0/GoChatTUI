package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
)

func connect(name, to string) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: "localhost:8123", Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	message := protocol.ClaimUsernameRequest{
		Username: name,
	}
	c.WriteJSON(&message)

	go (func() {
		for {
			_, message, err := c.ReadMessage()
			time.Sleep(time.Millisecond * 2)
			log.Print("recv:", name, string(message), err)
		}
	})()

	go (func() {
		i := 0
		for {
			message := protocol.ForwardMessageRequest{
				ToUsername: to,
				Content:    fmt.Sprintf("m %d", i),
			}
			i++
			c.WriteJSON(&message)
			time.Sleep(time.Millisecond * 2)
		}
	})()

	return c
}

func main() {
	_ = connect(os.Args[1], os.Args[2])

	for {

	}
}
