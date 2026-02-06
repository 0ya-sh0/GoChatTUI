package client

import (
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type ChatData struct {
	unread   int
	messages []protocol.Message
}

type UIState struct {
	username        string
	isMainScreen    bool
	unreadUsers     []string
	onlineUsers     []string
	offlineUsers    []string
	userPos         int
	messages        map[string]ChatData
	chosenTab       int
	height          int
	chosenUser      string
	activeUsers     map[string]bool
	currentChatData ChatData
	messageScroll   int
	currentText     string
}

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
			exit := false
			requireRender, exit = handleKeypress(state, event, conn)
			if exit {
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

func handleKeypress(state *UIState, event EventKeyPress, conn *websocket.Conn) (bool, bool) {
	requireRender := true
	exit := false
	if event.KeyType == KEY_TYPE_CTRL_C {
		if state.isMainScreen {
			return false, false
		}
		state.isMainScreen = true
		state.userPos = 0
		state.currentText = ""
	}
	/*

		available height for messages = height - FIXED
		number of messages = len(messages)
		scroll pos determines what first message is seen
		i.e if we want to show message i to i + (height-FIXED)
		scroll is i
		lowerbound i can be 0 (show first message)
		upperbound
			say we have height 20 and 5 messages, we cant scroll, i remains 0

			if we h = 20, m = 20, i remains 0
			h = 20, m = 21, i 0 or 1
			h = 20, m = 22, i 0, 1, 2

			h = 20, m = 100, i 0 to 80

			if h >= m, upperbound = 0
			else upperbound = m - (h - FIXED)

		or i can be min(len(messages), )


	*/
	if event.KeyType == KEY_TYPE_UP_ARROW {
		if state.isMainScreen {
			if state.chosenTab == 0 && len(state.unreadUsers) > 0 {
				state.userPos = (len(state.unreadUsers) + state.userPos - 1) % len(state.unreadUsers)
			}
			if state.chosenTab == 1 && len(state.onlineUsers) > 0 {
				state.userPos = (len(state.onlineUsers) + state.userPos - 1) % len(state.onlineUsers)
			}
			if state.chosenTab == 2 && len(state.offlineUsers) > 0 {
				state.userPos = (len(state.offlineUsers) + state.userPos - 1) % len(state.offlineUsers)
			}
		}
		if !state.isMainScreen && len(state.currentChatData.messages) > (state.height-FIXED) {
			if state.messageScroll > 0 {
				state.messageScroll--
			}
		}
	}
	if event.KeyType == KEY_TYPE_DOWN_ARROW {
		if state.isMainScreen {
			if state.chosenTab == 0 && len(state.unreadUsers) > 0 {
				state.userPos = (len(state.unreadUsers) + state.userPos + 1) % len(state.unreadUsers)
			}
			if state.chosenTab == 1 && len(state.onlineUsers) > 0 {
				state.userPos = (len(state.onlineUsers) + state.userPos + 1) % len(state.onlineUsers)
			}
			if state.chosenTab == 2 && len(state.offlineUsers) > 0 {
				state.userPos = (len(state.offlineUsers) + state.userPos + 1) % len(state.offlineUsers)
			}
		}
		if !state.isMainScreen && len(state.currentChatData.messages) > (state.height-FIXED) {
			if state.messageScroll < len(state.currentChatData.messages)-(state.height-FIXED) {
				state.messageScroll++
			}
		}
	}
	if event.KeyType == KEY_TYPE_LEFT_ARROW {
		if state.isMainScreen {
			state.chosenTab = (3 + state.chosenTab - 1) % 3
			state.userPos = 0
		}
	}
	if event.KeyType == KEY_TYPE_RIGHT_ARROW {
		if state.isMainScreen {
			state.chosenTab = (3 + state.chosenTab + 1) % 3
			state.userPos = 0
		}
	}

	if event.KeyType == KEY_TYPE_ENTER {
		if state.isMainScreen {
			if state.chosenTab == 0 && len(state.unreadUsers) > 0 {
				state.chosenUser = state.unreadUsers[state.userPos]
				state.isMainScreen = false
				data := state.messages[state.chosenUser]
				data.unread = 0
				state.messages[state.chosenUser] = data
				state.currentChatData = data
				// mark as read
				newUnread := make([]string, 0, len(state.unreadUsers)-1)
				for _, v := range state.unreadUsers {
					if v == state.chosenUser {
						continue
					}
					newUnread = append(newUnread, v)
				}
				state.unreadUsers = newUnread
				if state.activeUsers[state.chosenUser] {
					state.onlineUsers = append(state.onlineUsers, state.chosenUser)
				} else {
					state.offlineUsers = append(state.offlineUsers, state.chosenUser)
				}
			} else if state.chosenTab == 1 && len(state.onlineUsers) > 0 {
				state.chosenUser = state.onlineUsers[state.userPos]
				state.isMainScreen = false
				data := state.messages[state.chosenUser]
				data.unread = 0
				state.messages[state.chosenUser] = data
				state.currentChatData = data
			} else if state.chosenTab == 2 && len(state.offlineUsers) > 0 {
				state.chosenUser = state.offlineUsers[state.userPos]
				state.isMainScreen = false
				data := state.messages[state.chosenUser]
				data.unread = 0
				state.messages[state.chosenUser] = data
				state.currentChatData = data
			}
		}
		if !state.isMainScreen && len(state.currentText) > 0 {
			data := state.messages[state.chosenUser]
			localMessage := protocol.Message{
				FromUsername: state.username,
				ToUsername:   state.chosenUser,
				Content:      state.currentText,
				Timestamp:    time.Now(),
			}
			data.messages = append(data.messages, localMessage)
			state.messages[state.chosenUser] = data
			state.currentText = ""
			writeMessage(conn, localMessage)
			state.currentChatData = data
			if len(state.currentChatData.messages) > (state.height - FIXED) {
				state.messageScroll = len(state.currentChatData.messages) - (state.height - FIXED)
			}
		}

	}
	if event.KeyType == KEY_TYPE_PRINTABLE && !state.isMainScreen {
		if len(state.currentText) < 64 {
			state.currentText = state.currentText + string(event.Char)
		}
	}
	if event.KeyType == KEY_TYPE_BACKSPACE && !state.isMainScreen {
		if len(state.currentText) > 0 {
			bytes := []byte(state.currentText)
			bytes = bytes[:len(bytes)-1]
			state.currentText = string(bytes)
		}
	}
	return requireRender, exit
}

func handleWSMessage(state *UIState, event protocol.Message) bool {
	requireRender := false
	if event.Type == protocol.MESSAGE_TYPE_BROADCAST {
		requireRender = handleBroadcastMesasge(state, event)
	}
	if event.Type == protocol.MESSAGE_TYPE_CHAT {
		requireRender = handleChatMesasge(state, event)
	}
	return requireRender
}

func handleBroadcastMesasge(state *UIState, event protocol.Message) bool {
	requireRender := true
	state.unreadUsers = []string{}
	state.onlineUsers = []string{}
	state.offlineUsers = []string{}
	state.activeUsers = make(map[string]bool)
	if len(event.Users) > 1 {
		filtered := make([]string, 0, len(event.Users)-1)
		for _, v := range event.Users {
			if v == state.username {
				continue
			}
			if _, ok := state.messages[v]; !ok {
				state.messages[v] = ChatData{}
			}
			filtered = append(filtered, v)
			state.activeUsers[v] = true
		}
		for k, v := range state.messages {
			if v.unread > 0 {
				state.unreadUsers = append(state.unreadUsers, k)
			} else if state.activeUsers[k] {
				state.onlineUsers = append(state.onlineUsers, k)
			} else {
				state.offlineUsers = append(state.offlineUsers, k)
			}
		}
		sort.Strings(state.unreadUsers)
		sort.Strings(state.onlineUsers)
		sort.Strings(state.offlineUsers)
	}
	if state.chosenTab == 0 {
		if len(state.unreadUsers) == 0 {
			state.userPos = 0
		} else {
			state.userPos = min(state.userPos, len(state.unreadUsers)-1)
		}
	}
	if state.chosenTab == 1 {
		if len(state.onlineUsers) == 0 {
			state.userPos = 0
		} else {
			state.userPos = min(state.userPos, len(state.onlineUsers)-1)
		}
	}
	if state.chosenTab == 2 {
		if len(state.offlineUsers) == 0 {
			state.userPos = 0
		} else {
			state.userPos = min(state.userPos, len(state.offlineUsers)-1)
		}
	}
	return requireRender
}

func handleChatMesasge(state *UIState, event protocol.Message) bool {
	requireRender := true
	data := state.messages[event.FromUsername]
	data.messages = append(data.messages, protocol.Message{
		FromUsername: event.FromUsername,
		ToUsername:   event.ToUsername,
		Content:      event.Content,
		Timestamp:    event.Timestamp,
	})
	if state.isMainScreen || state.chosenUser != event.FromUsername {
		data.unread++
		state.messages[event.FromUsername] = data
	} else {
		state.messages[event.FromUsername] = data
		state.currentChatData = data
		if len(state.currentChatData.messages) > (state.height - FIXED) {
			state.messageScroll = len(state.currentChatData.messages) - (state.height - FIXED)
		}
	}

	state.unreadUsers = []string{}
	state.onlineUsers = []string{}
	state.offlineUsers = []string{}
	for k, v := range state.messages {
		if v.unread > 0 {
			state.unreadUsers = append(state.unreadUsers, k)
		} else if state.activeUsers[k] {
			state.onlineUsers = append(state.onlineUsers, k)
		} else {
			state.offlineUsers = append(state.offlineUsers, k)
		}
	}
	sort.Strings(state.unreadUsers)
	sort.Strings(state.onlineUsers)
	sort.Strings(state.offlineUsers)

	if state.chosenTab == 0 {
		if len(state.unreadUsers) == 0 {
			state.userPos = 0
		} else {
			state.userPos = min(state.userPos, len(state.unreadUsers)-1)
		}
	}
	if state.chosenTab == 1 {
		if len(state.onlineUsers) == 0 {
			state.userPos = 0
		} else {
			state.userPos = min(state.userPos, len(state.onlineUsers)-1)
		}
	}
	if state.chosenTab == 2 {
		if len(state.offlineUsers) == 0 {
			state.userPos = 0
		} else {
			state.userPos = min(state.userPos, len(state.offlineUsers)-1)
		}
	}
	return requireRender
}

func handleResize(state *UIState, newHeight int) bool {
	state.height = newHeight
	return true
}
