package client

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/0ya-sh0/GoChatTUI/internal/server"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

func connect(userName string, url url.URL) (*websocket.Conn, error) {
	log.Printf("connecting to %s", url.String())
	c, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return nil, err
	}
	message := server.UserNameMessage{
		UserName: userName,
	}
	err = c.WriteJSON(&message)
	if err != nil {
		return nil, err
	}
	return c, nil
}

const (
	KEY_TYPE_PRINTABLE = iota
	KEY_TYPE_CTRL_C
	KEY_TYPE_ESC
	KEY_TYPE_UP_ARROW
	KEY_TYPE_DOWN_ARROW
	KEY_TYPE_LEFT_ARROW
	KEY_TYPE_RIGHT_ARROW
	KEY_TYPE_ENTER
	KEY_TYPE_BACKSPACE
	KEY_TYPE_UNKNOWN
)

type KeyType int

type EventKeyPress struct {
	KeyType KeyType
	Char    byte
}

func listenWSEvents(events chan server.MessageToClient, conn *websocket.Conn) {
	for {
		event, err := readMessage(conn)
		if err != nil {
			close(events)
			break
		}
		events <- event
	}
}

func readMessage(conn *websocket.Conn) (server.MessageToClient, error) {
	message := server.MessageToClient{}
	err := conn.ReadJSON(&message)
	return message, err
}

func readKey() (EventKeyPress, error) {
	bt := make([]byte, 1)
	_, err := os.Stdin.Read(bt)
	if err != nil {
		return EventKeyPress{}, err
	}

	if isPrintable(bt[0]) {
		return EventKeyPress{
			KeyType: KEY_TYPE_PRINTABLE,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_CTRL_C {
		return EventKeyPress{
			KeyType: KEY_TYPE_CTRL_C,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_ENTER {
		return EventKeyPress{
			KeyType: KEY_TYPE_ENTER,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_BACKSPACE {
		return EventKeyPress{
			KeyType: KEY_TYPE_BACKSPACE,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_ESC {
		os.Stdin.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		defer os.Stdin.SetReadDeadline(time.Time{})

		seq := []byte{KEY_ESC}

		for {
			_, err := os.Stdin.Read(bt)
			if err != nil {
				return EventKeyPress{
					KeyType: KEY_TYPE_ESC,
					Char:    KEY_ESC,
				}, nil
			}
			seq = append(seq, bt[0])
			if len(seq) >= 3 {
				if string(seq) == string(KEY_UP) {
					return EventKeyPress{
						KeyType: KEY_TYPE_UP_ARROW,
					}, nil
				}

				if string(seq) == string(KEY_DOWN) {
					return EventKeyPress{
						KeyType: KEY_TYPE_DOWN_ARROW,
					}, nil
				}

				if string(seq) == string(KEY_LEFT) {
					return EventKeyPress{
						KeyType: KEY_TYPE_LEFT_ARROW,
					}, nil
				}

				if string(seq) == string(KEY_RIGHT) {
					return EventKeyPress{
						KeyType: KEY_TYPE_RIGHT_ARROW,
					}, nil
				}
				return EventKeyPress{
					KeyType: KEY_TYPE_UNKNOWN,
				}, nil
			}
		}
	}

	return EventKeyPress{
		KeyType: KEY_TYPE_UNKNOWN,
		Char:    bt[0],
	}, nil
}

func listenKeyEvents(events chan EventKeyPress) {
	for {
		event, err := readKey()
		if err != nil {
			close(events)
			break
		}
		events <- event
	}
}

const SEP = "──────────────────────────────────────────────────────"

func printHeader(userName string) {
	fmt.Print(Reset, SEP)
	fmt.Printf(CursorPos, 2, 1)
	fmt.Print(" GoChatTUI (v1.0) - Logged in as: ", userName, Reset)
	fmt.Printf(CursorPos, 3, 1)
	fmt.Print(Reset, SEP)
	fmt.Print(Reset)
}

func printUserName(userName string, activeUsers map[string]bool) {
	fmt.Printf(CursorPos, 4, 1)
	fmt.Print(" Chat with - ")
	if _, ok := activeUsers[userName]; ok {
		fmt.Print(FgGreen, " ● ", userName)
	} else {
		fmt.Print(Reset, " ○ ", userName)
	}
	fmt.Printf(CursorPos, 5, 1)
	fmt.Print(Reset, SEP)
}

func listenResizeEvents(events chan int) {
	_, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		close(events)
		return
	}
	for {
		time.Sleep(time.Second)
		_, newHeight, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			close(events)
			return
		}
		if newHeight != height {
			height = newHeight
			events <- height
		}
	}
}

func printCurrentText(currentText string, height int) {
	fmt.Printf(CursorPos, height-3, 1)
	fmt.Print(Reset, SEP)
	fmt.Printf(CursorPos, height-2, 1)
	if currentText == "" {
		fmt.Print(" > enter message... ")
	} else {
		fmt.Print(" > ", currentText)
	}
	fmt.Printf(CursorPos, height-1, 1)
	fmt.Print(Reset, SEP)
	fmt.Printf(CursorPos, height, 1)
	fmt.Print("↑ ↓ Scroll chat     Enter: Send     Ctrl+C: Back")
	fmt.Print(Reset)
}

const FIXED = 9

func printUsers(unreadUsers, onlineUsers, offlineUsers []string, userPos int, messages map[string]ChatData, chosenTab, height int) {
	line := 4
	fmt.Printf(CursorPos, line, 1)
	switch chosenTab {
	case 0:
		fmt.Print(" [ Unread ]   Online     Offline")
	case 1:
		fmt.Print("   Unread   [ Online ]   Offline")
	default:
		fmt.Print("   Unread     Online   [ Offline ]")
	}
	line = 5
	fmt.Printf(CursorPos, line, 1)
	fmt.Print(Reset, SEP)
	line = 6
	fmt.Printf(CursorPos, line, 1)

	if chosenTab == 0 {
		for pos, v := range unreadUsers {
			fmt.Print(Reset)
			if pos == userPos {
				fmt.Printf(CursorPos, line, 1)
				fmt.Print(Bold, "▶ ")
			} else {
				fmt.Printf(CursorPos, line, 3)
			}
			fmt.Print(v)
			fmt.Printf(CursorPos, line, 12)
			fmt.Print("(", messages[v].unread, ")")
			line++
		}
	}

	if chosenTab == 1 {
		for pos, v := range onlineUsers {
			fmt.Print(Reset)
			if pos == userPos {
				fmt.Printf(CursorPos, line, 1)
				fmt.Print(Bold, "▶ ")
			} else {
				fmt.Printf(CursorPos, line, 3)
			}
			line++
			fmt.Print(v)
		}
	}

	if chosenTab == 2 {
		for pos, v := range offlineUsers {
			fmt.Print(Reset)
			if pos == userPos {
				fmt.Printf(CursorPos, line, 1)
				fmt.Print(Bold, "▶ ")
			} else {
				fmt.Printf(CursorPos, line, 3)
			}
			line++
			fmt.Print(v)
		}
	}

	fmt.Printf(CursorPos, height, 1)
	fmt.Print(Reset, "← → Switch tabs     ↑ ↓ Move     Enter: Open     Ctrl+C: Quit")
}

func printMessages(userName, chosenUser string, data ChatData, height, messageScroll int) {
	line := 6
	start := messageScroll
	end := min(len(data.messages), messageScroll+(height-FIXED))
	unamePad := max(3, len(chosenUser)) - 3
	youPad := false
	if len(chosenUser) <= 3 {
		youPad = false
		unamePad = 3 - len(chosenUser)
	} else {
		youPad = true
		unamePad = len(chosenUser) - 3
	}

	for curr := start; curr < end; curr++ {
		v := data.messages[curr]
		fmt.Print(Reset)
		fmt.Printf(CursorPos, line, 1)
		line++
		v.Timestamp.Format(time.RFC822)
		if v.FromUserName == userName {
			fmt.Print(FgGreen, v.Timestamp.Format(time.DateOnly+" "+time.TimeOnly), " ", "you")
			if youPad {
				for i := 0; i < unamePad; i++ {
					fmt.Print(" ")
				}
			}
			fmt.Print(": ", Reset, v.Content)
		} else {
			fmt.Print(FgRed, v.Timestamp.Format(time.DateOnly+" "+time.TimeOnly), " ", v.FromUserName)
			if !youPad {
				for i := 0; i < unamePad; i++ {
					fmt.Print(" ")
				}
			}
			fmt.Print(": ", Reset, v.Content)
		}
	}
}

type Message struct {
	FromUserName string    `json:"fromUserName"`
	ToUserName   string    `json:"toUserName"`
	Content      string    `json:"content"`
	Timestamp    time.Time `json:"timestamp"`
}

type ChatData struct {
	unread   int
	messages []Message
}

func Start(userName string, url url.URL) error {
	conn, err := connect(userName, url)
	if err != nil {
		return err
	}
	defer conn.Close()

	keyEvents := make(chan EventKeyPress)
	wsEvents := make(chan server.MessageToClient)
	resizeEvents := make(chan int)
	messageScroll := 0
	messages := make(map[string]ChatData)
	unreadUsers := []string{}
	onlineUsers := []string{}
	offlineUsers := []string{}
	activeUsers := make(map[string]bool)
	userPos := 0
	isMainScreen := true
	chosenUser := ""
	currentText := ""
	currentChatData := ChatData{}
	chosenTab := 0
	_, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	go listenKeyEvents(keyEvents)
	go listenWSEvents(wsEvents, conn)
	go listenResizeEvents(resizeEvents)
	for {
		fmt.Print(ClearScreen, CursorHome, CursorHide)
		printHeader(userName)
		if isMainScreen {
			printUsers(unreadUsers, onlineUsers, offlineUsers, userPos, messages, chosenTab, height)
		} else {
			printUserName(chosenUser, activeUsers)
			printMessages(userName, chosenUser, currentChatData, height, messageScroll)
			printCurrentText(currentText, height)
		}
		select {
		case event, ok := <-keyEvents:
			if !ok {
				return nil
			}
			if event.KeyType == KEY_TYPE_CTRL_C {
				if isMainScreen {
					return nil
				}
				isMainScreen = true
				userPos = 0
				currentText = ""
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
				if isMainScreen {
					if chosenTab == 0 && len(unreadUsers) > 0 {
						userPos = (len(unreadUsers) + userPos - 1) % len(unreadUsers)
					}
					if chosenTab == 1 && len(onlineUsers) > 0 {
						userPos = (len(onlineUsers) + userPos - 1) % len(onlineUsers)
					}
					if chosenTab == 2 && len(offlineUsers) > 0 {
						userPos = (len(offlineUsers) + userPos - 1) % len(offlineUsers)
					}
				}
				if !isMainScreen && len(currentChatData.messages) > (height-FIXED) {
					if messageScroll > 0 {
						messageScroll--
					}
				}
			}
			if event.KeyType == KEY_TYPE_DOWN_ARROW {
				if isMainScreen {
					if chosenTab == 0 && len(unreadUsers) > 0 {
						userPos = (len(unreadUsers) + userPos + 1) % len(unreadUsers)
					}
					if chosenTab == 1 && len(onlineUsers) > 0 {
						userPos = (len(onlineUsers) + userPos + 1) % len(onlineUsers)
					}
					if chosenTab == 2 && len(offlineUsers) > 0 {
						userPos = (len(offlineUsers) + userPos + 1) % len(offlineUsers)
					}
				}
				if !isMainScreen && len(currentChatData.messages) > (height-FIXED) {
					if messageScroll < len(currentChatData.messages)-(height-FIXED) {
						messageScroll++
					}
				}
			}
			if event.KeyType == KEY_TYPE_LEFT_ARROW {
				if isMainScreen {
					chosenTab = (3 + chosenTab - 1) % 3
					userPos = 0
				}
			}
			if event.KeyType == KEY_TYPE_RIGHT_ARROW {
				if isMainScreen {
					chosenTab = (3 + chosenTab + 1) % 3
					userPos = 0
				}
			}

			if event.KeyType == KEY_TYPE_ENTER {
				if isMainScreen {
					if chosenTab == 0 && len(unreadUsers) > 0 {
						chosenUser = unreadUsers[userPos]
						isMainScreen = false
						data := messages[chosenUser]
						data.unread = 0
						messages[chosenUser] = data
						currentChatData = data
						// mark as read
						newUnread := make([]string, 0, len(unreadUsers)-1)
						for _, v := range unreadUsers {
							if v == chosenUser {
								continue
							}
							newUnread = append(newUnread, v)
						}
						unreadUsers = newUnread
						if activeUsers[chosenUser] {
							onlineUsers = append(onlineUsers, chosenUser)
						} else {
							offlineUsers = append(offlineUsers, chosenUser)
						}
					} else if chosenTab == 1 && len(onlineUsers) > 0 {
						chosenUser = onlineUsers[userPos]
						isMainScreen = false
						data := messages[chosenUser]
						data.unread = 0
						messages[chosenUser] = data
						currentChatData = data
					} else if chosenTab == 2 && len(offlineUsers) > 0 {
						chosenUser = offlineUsers[userPos]
						isMainScreen = false
						data := messages[chosenUser]
						data.unread = 0
						messages[chosenUser] = data
						currentChatData = data
					}
				}
				if !isMainScreen && len(currentText) > 0 {
					data := messages[chosenUser]
					localMessage := Message{
						FromUserName: userName,
						ToUserName:   chosenUser,
						Content:      currentText,
						Timestamp:    time.Now(),
					}
					data.messages = append(data.messages, localMessage)
					messages[chosenUser] = data
					currentText = ""
					conn.WriteJSON(localMessage)
					currentChatData = data
					if len(currentChatData.messages) > (height - FIXED) {
						messageScroll = len(currentChatData.messages) - (height - FIXED)
					}
				}

			}
			if event.KeyType == KEY_TYPE_PRINTABLE && !isMainScreen {
				if len(currentText) < 64 {
					currentText = currentText + string(event.Char)
				}
			}
			if event.KeyType == KEY_TYPE_BACKSPACE && !isMainScreen {
				if len(currentText) > 0 {
					bytes := []byte(currentText)
					bytes = bytes[:len(bytes)-1]
					currentText = string(bytes)
				}
			}
		case event, ok := <-wsEvents:
			if !ok {
				return nil
			}
			if event.Type == server.BROADCAST_TYPE {
				unreadUsers = []string{}
				onlineUsers = []string{}
				offlineUsers = []string{}
				activeUsers = make(map[string]bool)
				if len(event.Users) > 1 {
					filtered := make([]string, 0, len(event.Users)-1)
					for _, v := range event.Users {
						if v == userName {
							continue
						}
						if _, ok := messages[v]; !ok {
							messages[v] = ChatData{}
						}
						filtered = append(filtered, v)
						activeUsers[v] = true
					}
					for k, v := range messages {
						if v.unread > 0 {
							unreadUsers = append(unreadUsers, k)
						} else if activeUsers[k] {
							onlineUsers = append(onlineUsers, k)
						} else {
							offlineUsers = append(offlineUsers, k)
						}
					}
					sort.Strings(unreadUsers)
					sort.Strings(onlineUsers)
					sort.Strings(offlineUsers)
				}
				if chosenTab == 0 {
					if len(unreadUsers) == 0 {
						userPos = 0
					} else {
						userPos = min(userPos, len(unreadUsers)-1)
					}
				}
				if chosenTab == 1 {
					if len(onlineUsers) == 0 {
						userPos = 0
					} else {
						userPos = min(userPos, len(onlineUsers)-1)
					}
				}
				if chosenTab == 2 {
					if len(offlineUsers) == 0 {
						userPos = 0
					} else {
						userPos = min(userPos, len(offlineUsers)-1)
					}
				}
			}
			if event.Type == server.CHAT_TYPE {
				data := messages[event.FromUserName]
				data.messages = append(data.messages, Message{
					FromUserName: event.FromUserName,
					ToUserName:   userName,
					Content:      event.Content,
					Timestamp:    event.Timestamp,
				})
				if isMainScreen || chosenUser != event.FromUserName {
					data.unread++
					messages[event.FromUserName] = data
				} else {
					messages[event.FromUserName] = data
					currentChatData = data
					if len(currentChatData.messages) > (height - FIXED) {
						messageScroll = len(currentChatData.messages) - (height - FIXED)
					}
				}

				unreadUsers = []string{}
				onlineUsers = []string{}
				offlineUsers = []string{}
				for k, v := range messages {
					if v.unread > 0 {
						unreadUsers = append(unreadUsers, k)
					} else if activeUsers[k] {
						onlineUsers = append(onlineUsers, k)
					} else {
						offlineUsers = append(offlineUsers, k)
					}
				}
				sort.Strings(unreadUsers)
				sort.Strings(onlineUsers)
				sort.Strings(offlineUsers)

				if chosenTab == 0 {
					if len(unreadUsers) == 0 {
						userPos = 0
					} else {
						userPos = min(userPos, len(unreadUsers)-1)
					}
				}
				if chosenTab == 1 {
					if len(onlineUsers) == 0 {
						userPos = 0
					} else {
						userPos = min(userPos, len(onlineUsers)-1)
					}
				}
				if chosenTab == 2 {
					if len(offlineUsers) == 0 {
						userPos = 0
					} else {
						userPos = min(userPos, len(offlineUsers)-1)
					}
				}
			}
		case event, ok := <-resizeEvents:
			if !ok {
				return nil
			}
			height = event
		}
	}
}
