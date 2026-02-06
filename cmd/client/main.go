package main

import (
	"log"
	"net/url"
	"os"

	"github.com/0ya-sh0/GoChatTUI/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Username is required: $ ./client user123")
		return
	}
	url := url.URL{Scheme: "ws", Host: "localhost:8123", Path: "/ws"}
	username := os.Args[1]

	if err := client.SetupTerminal(); err != nil {
		log.Fatal(err)
		return
	}
	defer client.RestoreTerminal()

	if err := client.Start(username, url); err != nil {
		log.Fatal(err)
	}
}
