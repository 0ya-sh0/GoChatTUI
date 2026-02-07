package client

import (
	"net/url"
	"os"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"golang.org/x/term"
)

func Start(username string, url url.URL) error {
	_, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	conn, err := connect(username, url)
	if err != nil {
		return err
	}
	defer conn.Close()

	keyEvents := make(chan EventKeyPress)
	wsEvents := make(chan protocol.Message)
	resizeEvents := make(chan int)
	go listenKeyEvents(keyEvents)
	go listenWSEvents(conn, wsEvents)
	go listenResizeEvents(resizeEvents)

	state := &UIState{
		username:        username,
		conn:            conn,
		unreadUsers:     []string{},
		onlineUsers:     []string{},
		offlineUsers:    []string{},
		messageScroll:   0,
		messages:        make(map[string]ChatData),
		activeUsers:     make(map[string]bool),
		userPos:         0,
		isMainScreen:    true,
		chosenUser:      "",
		currentText:     "",
		currentChatData: ChatData{},
		chosenTab:       0,
		height:          h,
		exit:            false,
	}
	requireRender := true
	for {
		if requireRender {
			render(state)
			requireRender = false
		}
		select {
		case event, ok := <-keyEvents:
			if !ok {
				return nil
			}
			requireRender = handleKeypress(state, event)
			if state.exit {
				return nil
			}
		case event, ok := <-wsEvents:
			if !ok {
				return nil
			}
			requireRender = handleWSMessage(state, event)
		case newHeight, ok := <-resizeEvents:
			if !ok {
				return nil
			}
			requireRender = handleResize(state, newHeight)
		}
	}
}
