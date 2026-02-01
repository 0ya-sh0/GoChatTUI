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

func printHeader(userName string) {
	fmt.Print(Reset, Bold, "----------")
	fmt.Printf(CursorPos, 2, 1)
	fmt.Print(FgRed, "GoChatTUI (v1.0) - ", userName, Reset)
	fmt.Printf(CursorPos, 3, 1)
	fmt.Print(Bold, "----------")
	fmt.Print(Reset)
}

func printUserName(userName string, activeUsers map[string]bool) {
	fmt.Printf(CursorPos, 4, 1)
	fmt.Print("(<- CTRL-C)   ")
	if _, ok := activeUsers[userName]; ok {
		fmt.Print(FgGreen, " ● ")
	} else {
		fmt.Print(FgGreen, " ○ ")
	}
	fmt.Print(userName)
	fmt.Printf(CursorPos, 5, 1)
	fmt.Print("----------")
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
	fmt.Printf(CursorPos, height-1, 1)
	fmt.Print(Reset, "----------")
	fmt.Printf(CursorPos, height, 1)
	if currentText == "" {
		fmt.Print(BgBlack, FgMagenta, "Enter text... (Enter) to send")
	} else {
		fmt.Print(BgBlack, FgWhite, currentText)
	}
	fmt.Print(Reset)
}

func printUsers(users []string, userPos int, messages map[string]ChatData) {
	i := 6
	for pos, v := range users {
		fmt.Print(Reset)
		if pos == userPos {
			fmt.Printf(CursorPos, i, 1)
			fmt.Print(Bold, BgBlack, FgRed, "▶ ")
		} else {
			fmt.Printf(CursorPos, i, 3)
		}
		i++
		fmt.Print(FgRed, v, " (", messages[v].unread, ")")
	}
	fmt.Print(Reset)
}

func printMessages(userName string, data ChatData, height, messageScroll int) {
	line := 6
	start := messageScroll
	end := min(len(data.messages), messageScroll+(height-7))
	for curr := start; curr < end; curr++ {
		v := data.messages[curr]
		fmt.Print(Reset)
		fmt.Printf(CursorPos, line, 1)
		line++
		v.Timestamp.Format(time.RFC822)
		if v.FromUserName == userName {
			fmt.Print(FgGreen, v.Timestamp.Format(time.DateOnly+" "+time.TimeOnly), " ", "(You): ", Reset, v.Content)
		} else {
			fmt.Print(FgRed, v.Timestamp.Format(time.DateOnly+" "+time.TimeOnly), " ", v.FromUserName, ": ", Reset, v.Content)
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
	users := []string{}
	activeUsers := make(map[string]bool)
	userPos := 0
	isMainScreen := true
	chosenUser := ""
	currentText := ""
	currentChatData := ChatData{}
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
			printUsers(users, userPos, messages)
		} else {
			printUserName(chosenUser, activeUsers)
			printMessages(userName, currentChatData, height, messageScroll)
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

				available height for messages = height - 7
				number of messages = len(messages)
				scroll pos determines what first message is seen
				i.e if we want to show message i to i + (height-7)
				scroll is i
				lowerbound i can be 0 (show first message)
				upperbound
					say we have height 20 and 5 messages, we cant scroll, i remains 0

					if we h = 20, m = 20, i remains 0
					h = 20, m = 21, i 0 or 1
					h = 20, m = 22, i 0, 1, 2

					h = 20, m = 100, i 0 to 80

					if h >= m, upperbound = 0
					else upperbound = m - (h - 7)

				or i can be min(len(messages), )


			*/
			if event.KeyType == KEY_TYPE_UP_ARROW {
				if isMainScreen && len(users) > 0 {
					userPos = (len(users) + userPos - 1) % len(users)
				}
				if !isMainScreen && len(currentChatData.messages) > (height-7) {
					if messageScroll > 0 {
						messageScroll--
					}
				}
			}
			if event.KeyType == KEY_TYPE_DOWN_ARROW {
				if isMainScreen && len(users) > 0 {
					userPos = (len(users) + userPos + 1) % len(users)
				}
				if !isMainScreen && len(currentChatData.messages) > (height-7) {
					if messageScroll < len(currentChatData.messages)-(height-7) {
						messageScroll++
					}
				}
			}
			if event.KeyType == KEY_TYPE_ENTER {
				if isMainScreen && len(users) > 0 {
					chosenUser = users[userPos]
					isMainScreen = false
					data := messages[chosenUser]
					data.unread = 0
					messages[chosenUser] = data
					currentChatData = data
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
					if len(currentChatData.messages) > (height - 7) {
						messageScroll = len(currentChatData.messages) - (height - 7)
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
				activeUsers = make(map[string]bool)
				users = event.Users
				if len(users) <= 1 {
					users = []string{}
				} else {
					filtered := make([]string, 0, len(users)-1)
					for _, v := range users {
						if v == userName {
							continue
						}
						if _, ok := messages[v]; !ok {
							messages[v] = ChatData{}
						}
						filtered = append(filtered, v)
						activeUsers[v] = true
					}
					users = filtered
					sort.Strings(users)
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
					if len(currentChatData.messages) > (height - 7) {
						messageScroll = len(currentChatData.messages) - (height - 7)
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
