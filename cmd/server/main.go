package main

import (
	"log"
	"net/http"

	"github.com/0ya-sh0/GoChatTUI/internal/server"
)

func main() {
	http.HandleFunc("/ws", server.ProcessWsConnect)
	go server.ProcessNoNameConns()
	go server.ProcessJoinUser()
	go server.ProcessSendChatMessage()
	go server.ProcessCloseUser()
	log.Print("start on localhost:8123")
	http.ListenAndServe("localhost:8123", nil)
}
