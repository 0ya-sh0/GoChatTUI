package client

import (
	"net/url"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
)

func Start(username string, url url.URL) error {
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

	state := NewUIState(username, conn)
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
