package main

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/0ya-sh0/GoChatTUI/internal/client"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Username is required: $ ./client user123")
		return
	}
	url := url.URL{Scheme: "ws", Host: "localhost:8123", Path: "/ws"}
	username := os.Args[1]
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	fmt.Print(client.ANSI_ENTER_ALT_SCREEN)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	defer (func() {
		fmt.Print(client.ClearScreen, client.CursorHome, client.CursorShow)
		fmt.Print(client.ANSI_EXIT_ALT_SCREEN)
	})()
	if err := client.Start(username, url); err != nil {
		log.Fatal(err)
	}
}
