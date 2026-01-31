package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{}
var state = NewServerState()
var noNameConnChan chan *websocket.Conn = make(chan *websocket.Conn, 1024)
var joinUserChan chan JoinUserRequest = make(chan JoinUserRequest, 1024)
var sendChatChan chan MessageToClient = make(chan MessageToClient, 1024)
var closeUserChan chan string = make(chan string, 1024)

func ProcessWsConnect(w http.ResponseWriter, r *http.Request) {
	log.Print("req connection")
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("err upgrade:", err)
		return
	}
	log.Print("req connection waiting for username req")
	noNameConnChan <- conn
}

func ProcessNoNameConns() {
	log.Print("infinite ProcessNoNameConns")
	for {
		conn, ok := <-noNameConnChan
		if !ok {
			log.Print("cant read")
		}
		log.Print("waiting for conn1")
		go processNoNameConn(conn)
	}
}

func processNoNameConn(conn *websocket.Conn) {
	log.Print("waiting for conn2")
	nameChan := make(chan *UserNameMessage)
	defer close(nameChan)

	go (func() {
		var userNameModel UserNameMessage
		err := conn.ReadJSON(&userNameModel)
		if err == nil {
			nameChan <- &userNameModel
		} else {
			log.Print("err username:", err)
			nameChan <- nil
		}
	})()

	select {
	case <-time.After(time.Second * 5):
		conn.Close()
		log.Print("closing client no username requested")
		return
	case userNameOrNil := <-nameChan:
		if userNameOrNil == nil {
			conn.Close()
			return
		}
		joinUserChan <- JoinUserRequest{
			UserName: userNameOrNil.UserName,
			conn:     conn,
		}
	}
}

func ProcessJoinUser() {
	log.Print("infinite ProcessJoinUser")

	for {
		request := <-joinUserChan
		if joined := state.AddUser(request); joined {
			log.Print("success join:", request.UserName)
			state.BroadCast()
		} else {
			request.conn.Close()
		}
	}
}

func ProcessSendChatMessage() {
	log.Print("infinite ProcessSendChatMessage")

	for {
		message := <-sendChatChan
		state.SendMessage(message.ToUserName, message)
	}
}

type ServerState struct {
	users      map[string]JoinedUser
	usersMutex sync.Mutex
}

func NewServerState() *ServerState {
	return &ServerState{
		users:      map[string]JoinedUser{},
		usersMutex: sync.Mutex{},
	}
}

func (s *ServerState) AddUser(req JoinUserRequest) bool {
	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()

	if _, has := s.users[req.UserName]; has {
		return false
	}
	joinedUser := JoinedUser{
		UserName:        req.UserName,
		conn:            req.conn,
		sendMessageChan: make(chan interface{}, 1024),
	}
	s.users[req.UserName] = joinedUser

	go (func() {
		for {
			message, ok := <-joinedUser.sendMessageChan
			if !ok {
				closeUserChan <- joinedUser.UserName
				break
			}
			err := joinedUser.conn.WriteJSON(message)
			if err != nil {
				closeUserChan <- joinedUser.UserName
				break
			}
		}
	})()
	go (func() {
		for {
			var message ChatMessageRequest
			err := joinedUser.conn.ReadJSON(&message)
			if err != nil {
				log.Print("unhandled incorrect message (conn close read): ", joinedUser.UserName)
				closeUserChan <- joinedUser.UserName
				break
			}
			sendChatChan <- MessageToClient{
				Type:         CHAT_TYPE,
				FromUserName: joinedUser.UserName,
				ToUserName:   message.ToUserName,
				Content:      message.Content,
				Timestamp:    time.Now(),
			}
		}
	})()
	return true
}

func (s *ServerState) SendMessage(userName string, message interface{}) {
	s.usersMutex.Lock()
	if user, has := s.users[userName]; !has {
		s.usersMutex.Unlock()
	} else {
		s.usersMutex.Unlock()
		user.sendMessageChan <- message
	}
}

func (s *ServerState) BroadCast() {
	s.usersMutex.Lock()
	keys := make([]string, 0, len(s.users))
	chans := make([]chan interface{}, 0, len(s.users))

	for k, v := range s.users {
		keys = append(keys, k)
		chans = append(chans, v.sendMessageChan)
	}
	s.usersMutex.Unlock()

	message := MessageToClient{
		Type:  BROADCAST_TYPE,
		Users: keys,
	}

	for _, v := range chans {
		v <- message
	}
}

func (s *ServerState) RemoveUser(userName string) bool {
	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()
	if val, ok := s.users[userName]; ok {
		delete(s.users, userName)
		val.conn.Close()
		close(val.sendMessageChan)
		return true
	}
	return false
}

func ProcessCloseUser() {
	for {
		user := <-closeUserChan
		if state.RemoveUser(user) {
			state.BroadCast()
		}
	}
}

const CHAT_TYPE = "CHAT"
const BROADCAST_TYPE = "BROADCAST"

type UserNameMessage struct {
	UserName string `json:"userName"`
}

type JoinUserRequest struct {
	UserName string
	conn     *websocket.Conn
}

type JoinedUser struct {
	UserName        string
	conn            *websocket.Conn
	sendMessageChan chan interface{}
}

type ChatMessageRequest struct {
	ToUserName string `json:"toUserName"`
	Content    string `json:"content"`
}

type MessageToClient struct {
	Type         string    `json:"type"`
	FromUserName string    `json:"fromUserName"`
	ToUserName   string    `json:"toUserName"`
	Content      string    `json:"content"`
	Timestamp    time.Time `json:"timestamp"`
	Users        []string  `json:"users"`
}
