package main

import (
	"log"
	"net/http"

	"github.com/0ya-sh0/GoChatTUI/internal/server"
)

func main() {
	broker := server.NewBroker()
	broker.Start()
	http.HandleFunc("/ws", broker.ProcessWsConnect)
	log.Print("start on localhost:8123")
	http.ListenAndServe("localhost:8123", nil)
}
