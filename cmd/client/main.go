package main

import (
	"log"
	"net/url"
	"os"

	"github.com/0ya-sh0/GoChatTUI/internal/client"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Username and server is required: $ ./client localhost:8123 user123")
		return
	}
	host := os.Args[1]
	url := url.URL{Scheme: "ws", Host: host, Path: "/ws"}
	username := os.Args[2]

	if err := client.SetupTerminal(); err != nil {
		log.Fatal(err)
		return
	}
	defer client.RestoreTerminal()

	if err := client.Start(username, url); err != nil {
		log.Fatal(err)
	}
}
