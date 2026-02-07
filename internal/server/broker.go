package server

import (
	"log"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
)

type JoinUserRequest struct {
	username string
	conn     *websocket.Conn
}

type User struct {
	username   string
	conn       *websocket.Conn
	messageBox chan interface{}
}

type Broker struct {
	users               map[string]User
	joinUserRequests    chan JoinUserRequest
	stop                chan struct{}
	messageBroker       chan protocol.Message
	kickOutUserRequests chan string
	connectionRequests  chan *websocket.Conn
}

func NewBroker() *Broker {
	return &Broker{
		users:               make(map[string]User),
		joinUserRequests:    make(chan JoinUserRequest, 1024),
		kickOutUserRequests: make(chan string, 1024),
		messageBroker:       make(chan protocol.Message, 1024),
		stop:                make(chan struct{}),
		connectionRequests:  make(chan *websocket.Conn, 1024),
	}
}

func (b *Broker) handleStop() {
	for _, val := range b.users {
		val.conn.Close()
		close(val.messageBox)
	}
	close(b.joinUserRequests)
	close(b.kickOutUserRequests)
	close(b.messageBroker)
	close(b.stop)
	close(b.connectionRequests)
}

func (b *Broker) Start() {
	go func() {
		for {
			select {
			case joinUserRequest := <-b.joinUserRequests:
				b.handleJoinUserRequest(joinUserRequest)
			case message := <-b.messageBroker:
				b.handleMessageForwarding(message)
			case username := <-b.kickOutUserRequests:
				b.handleKickOutUser(username)
			case _ = <-b.stop:
				b.handleStop()
				return
			}
		}
	}()
}

func (b *Broker) Stop() {
	b.stop <- struct{}{}
}

func (b *Broker) handleJoinUserRequest(request JoinUserRequest) {
	if _, has := b.users[request.username]; has {
		request.conn.Close()
	} else {
		log.Print("join user: ", request.username)
		messageBox := make(chan interface{}, 1024)
		joinedUser := User{
			username:   request.username,
			conn:       request.conn,
			messageBox: messageBox,
		}
		b.users[request.username] = joinedUser
		go messageReciever(request.conn, b.messageBroker, request.username, b.kickOutUserRequests)
		go messageSender(request.conn, messageBox)
		b.broadcast()
	}
}

func (b *Broker) handleMessageForwarding(message protocol.Message) {
	if user, has := b.users[message.ToUsername]; has {
		user.messageBox <- message
	}
}

func (b *Broker) broadcast() {
	keys := make([]string, 0, len(b.users))
	for k := range b.users {
		keys = append(keys, k)
	}
	message := protocol.Message{
		Type:  protocol.MESSAGE_TYPE_BROADCAST,
		Users: keys,
	}
	for _, user := range b.users {
		user.messageBox <- message
	}
}

func (b *Broker) handleKickOutUser(username string) {
	if user, has := b.users[username]; has {
		close(user.messageBox)
		delete(b.users, username)
		b.broadcast()
		log.Print("kick user: ", username)
	}
}
