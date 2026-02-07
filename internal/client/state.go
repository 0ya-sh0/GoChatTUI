package client

import (
	"os"
	"sort"
	"time"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type UIState struct {
	username        string
	conn            *websocket.Conn
	isMainScreen    bool
	unreadUsers     []string
	onlineUsers     []string
	offlineUsers    []string
	userPos         int
	chats           map[string]ChatData
	chosenTab       int
	height          int
	chosenUser      string
	activeUsers     map[string]bool
	currentChatData ChatData
	messageScroll   int
	currentText     string
	exit            bool
}

func NewUIState(username string, conn *websocket.Conn) *UIState {
	_, h, _ := term.GetSize(int(os.Stdin.Fd()))
	persistedState := loadState(username)
	state := &UIState{
		username:        username,
		conn:            conn,
		unreadUsers:     []string{},
		onlineUsers:     []string{},
		offlineUsers:    []string{},
		messageScroll:   0,
		chats:           persistedState.Chats,
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
	return state
}

func handleResize(state *UIState, newHeight int) bool {
	state.height = newHeight
	return true
}

func handleWSMessage(state *UIState, event protocol.Message) bool {
	requireRender := false
	if event.Type == protocol.MESSAGE_TYPE_BROADCAST {
		requireRender = handleBroadcastMesasge(state, event)
	}
	if event.Type == protocol.MESSAGE_TYPE_CHAT {
		requireRender = handleChatMesasge(state, event)
	}
	writeState(PersistedState{
		Username: state.username, Chats: state.chats,
	})
	return requireRender
}

func updateTabLists(state *UIState) {
	state.unreadUsers = []string{}
	state.onlineUsers = []string{}
	state.offlineUsers = []string{}
	for k, v := range state.chats {
		if v.Unread > 0 {
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

func updateUserPos(state *UIState, delta int) bool {
	currentListLen := 0
	switch state.chosenTab {
	case 0:
		currentListLen = len(state.unreadUsers)
	case 1:
		currentListLen = len(state.onlineUsers)
	case 2:
		currentListLen = len(state.offlineUsers)
	}
	if currentListLen == 0 {
		state.userPos = 0
	} else {
		if delta == 0 {
			state.userPos = min(state.userPos, currentListLen-1)
		} else {
			state.userPos = (currentListLen + state.userPos + delta) % currentListLen
		}
	}
	return true
}

/*
available height for messages = height - FIXED
number of messages = len(messages)
scroll position determines which first message is seen.
i.e., if we want to show message i to i + (height - FIXED)
scroll is i.
Lower bound for i can be 0 (show first message).
Upper bound:

	Say we have height 20 and 5 messages; we can't scroll, so i remains 0.

	If we have h = 20, m = 20, i remains 0.
	If h = 20, m = 21, i can be 0 or 1.
	If h = 20, m = 22, i can be 0, 1, or 2.

	If h = 20, m = 100, i can range from 0 to 80.

If h >= m, upper bound = 0.
Otherwise, upper bound = m - (h - FIXED).
*/
func updateChatScroll(state *UIState, delta int) bool {
	messageListHeight := state.height - FIXED
	maximumStart := len(state.currentChatData.Messages) - messageListHeight
	prevMessageScroll := state.messageScroll
	if len(state.currentChatData.Messages) > messageListHeight {
		if delta == 0 {
			state.messageScroll = maximumStart
		} else {
			state.messageScroll = state.messageScroll + delta
			state.messageScroll = max(0, state.messageScroll)
			state.messageScroll = min(state.messageScroll, maximumStart)
		}
	}
	return prevMessageScroll != state.messageScroll
}

func updateChosenTabAndUserPos(state *UIState, delta int) bool {
	TAB_COUNT := 3
	if state.isMainScreen {
		state.chosenTab = (TAB_COUNT + state.chosenTab + delta) % TAB_COUNT
		state.userPos = 0
	}
	return state.isMainScreen
}

func handleChatMesasge(state *UIState, event protocol.Message) bool {
	requireRender := true
	data := state.chats[event.FromUsername]
	data.Messages = append(data.Messages, protocol.Message{
		FromUsername: event.FromUsername,
		ToUsername:   event.ToUsername,
		Content:      event.Content,
		Timestamp:    event.Timestamp,
	})

	if state.isMainScreen && state.chosenTab != 0 {
		requireRender = false
	}

	if !state.isMainScreen && state.chosenUser != event.FromUsername {
		requireRender = false
	}

	if state.isMainScreen || state.chosenUser != event.FromUsername {
		data.Unread++
		state.chats[event.FromUsername] = data
	} else {
		state.chats[event.FromUsername] = data
		state.currentChatData = data
		updateChatScroll(state, 0)
	}

	updateTabLists(state)
	updateUserPos(state, 0)

	return requireRender
}

func handleBroadcastMesasge(state *UIState, event protocol.Message) bool {
	requireRender := true
	state.activeUsers = make(map[string]bool)
	if len(event.Users) > 1 {
		filtered := make([]string, 0, len(event.Users)-1)
		for _, v := range event.Users {
			if v == state.username {
				continue
			}
			if _, ok := state.chats[v]; !ok {
				state.chats[v] = ChatData{}
			}
			filtered = append(filtered, v)
			state.activeUsers[v] = true
		}
	}
	updateTabLists(state)
	updateUserPos(state, 0)
	return requireRender
}

func handlePrintableKey(state *UIState, event EventKeyPress) bool {
	if !state.isMainScreen && len(state.currentText) < 64 {
		state.currentText = state.currentText + string(event.Char)
		return true
	}
	return false
}

func handleBackspace(state *UIState) bool {
	if !state.isMainScreen && len(state.currentText) > 0 {
		bytes := []byte(state.currentText)
		bytes = bytes[:len(bytes)-1]
		state.currentText = string(bytes)
		return true
	}
	return false
}

func handleCtrlC(state *UIState) bool {
	if state.isMainScreen {
		state.exit = true
		return false
	}
	state.isMainScreen = true
	state.userPos = 0
	state.currentText = ""
	return true
}

func handleLeftArrow(state *UIState) bool {
	return updateChosenTabAndUserPos(state, -1)
}

func handleRightArrow(state *UIState) bool {
	return updateChosenTabAndUserPos(state, 1)
}

func handleUpArrow(state *UIState) bool {
	if state.isMainScreen {
		return updateUserPos(state, -1)
	} else {
		return updateChatScroll(state, -1)
	}
}

func handleDownArrow(state *UIState) bool {
	if state.isMainScreen {
		return updateUserPos(state, 1)
	} else {
		return updateChatScroll(state, 1)
	}
}

func handleKeypress(state *UIState, event EventKeyPress) bool {
	switch event.KeyType {
	case KEY_TYPE_CTRL_C:
		return handleCtrlC(state)
	case KEY_TYPE_UP_ARROW:
		return handleUpArrow(state)
	case KEY_TYPE_DOWN_ARROW:
		return handleDownArrow(state)
	case KEY_TYPE_LEFT_ARROW:
		return handleLeftArrow(state)
	case KEY_TYPE_RIGHT_ARROW:
		return handleRightArrow(state)
	case KEY_TYPE_ENTER:
		return handleEnter(state)
	case KEY_TYPE_PRINTABLE:
		return handlePrintableKey(state, event)
	case KEY_TYPE_BACKSPACE:
		return handleBackspace(state)
	}
	return false
}

func handleEnter(state *UIState) bool {
	if state.isMainScreen {
		chosenList := []string{}
		switch state.chosenTab {
		case 0:
			chosenList = state.unreadUsers
		case 1:
			chosenList = state.onlineUsers
		case 2:
			chosenList = state.offlineUsers
		default:
			return false
		}
		if len(chosenList) == 0 {
			return false
		}
		state.chosenUser = chosenList[state.userPos]
		state.isMainScreen = false
		data := state.chats[state.chosenUser]
		data.Unread = 0
		state.chats[state.chosenUser] = data
		state.currentChatData = data
		if state.chosenTab == 0 {
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
		}
		return true
	}
	if len(state.currentText) == 0 {
		return false
	}
	data := state.chats[state.chosenUser]
	localMessage := protocol.Message{
		FromUsername: state.username,
		ToUsername:   state.chosenUser,
		Content:      state.currentText,
		Timestamp:    time.Now(),
	}
	data.Messages = append(data.Messages, localMessage)
	state.chats[state.chosenUser] = data
	state.currentText = ""
	if err := writeMessage(state.conn, localMessage); err != nil {
		state.conn.Close()
		state.exit = true
		return false
	}
	writeState(PersistedState{
		Username: state.username, Chats: state.chats,
	})
	state.currentChatData = data
	updateChatScroll(state, 0)
	return true
}
