package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Broker struct {
	users             map[string]JoinedUser
	joinUserRequests  chan JoinUserRequest
	stop              chan struct{}
	sendChatChan      chan MessageToClient
	leaveUserRequests chan string
	noNameConnChan    chan *websocket.Conn
}

func NewBroker() *Broker {
	return &Broker{
		users:             make(map[string]JoinedUser),
		joinUserRequests:  make(chan JoinUserRequest, 1024),
		leaveUserRequests: make(chan string, 1024),
		sendChatChan:      make(chan MessageToClient, 1024),
		stop:              make(chan struct{}),
		noNameConnChan:    make(chan *websocket.Conn, 1024),
	}
}

func (b *Broker) Start() {
	go func() {
	eventLoop:
		for {
			select {
			case joinUserRequest := <-b.joinUserRequests:
				if _, has := b.users[joinUserRequest.UserName]; has {
					joinUserRequest.conn.Close()
				} else {
					sendMessageChan := make(chan interface{}, 1024)
					joinedUser := JoinedUser{
						UserName:        joinUserRequest.UserName,
						conn:            joinUserRequest.conn,
						sendMessageChan: sendMessageChan,
					}
					b.users[joinUserRequest.UserName] = joinedUser
					go func(conn *websocket.Conn, outbox chan<- MessageToClient, userName string, leaveUser chan<- string) {
						for {
							var message ChatMessageRequest
							err := conn.ReadJSON(&message)
							if err != nil {
								conn.Close()
								leaveUser <- userName
								break
							}
							log.Print(userName, message)
							outbox <- MessageToClient{
								Type:         CHAT_TYPE,
								FromUserName: userName,
								ToUserName:   message.ToUserName,
								Content:      message.Content,
								Timestamp:    time.Now(),
							}
						}
					}(joinUserRequest.conn, b.sendChatChan, joinUserRequest.UserName, b.leaveUserRequests)

					go func(conn *websocket.Conn, inbox <-chan interface{}) {
						draining := false
						for {
							message, ok := <-inbox
							if !ok {
								return
							}
							if draining {
								continue
							}
							conn.SetWriteDeadline(time.Now().Add(time.Second))
							err := conn.WriteJSON(message)
							if err != nil {
								draining = true
								conn.Close()
							}
						}
					}(joinUserRequest.conn, sendMessageChan)
					b.broadcast()
				}
			case sendMessage := <-b.sendChatChan:
				log.Print(sendMessage)
				if user, has := b.users[sendMessage.ToUserName]; has {
					user.sendMessageChan <- sendMessage
				}
			case leaveUserRequest := <-b.leaveUserRequests:
				if user, has := b.users[leaveUserRequest]; has {
					close(user.sendMessageChan)
					delete(b.users, leaveUserRequest)
					b.broadcast()
				}
			case _ = <-b.stop:
				break eventLoop
			}
		}
		for _, val := range b.users {
			val.conn.Close()
			close(val.sendMessageChan)
		}
		close(b.joinUserRequests)
		close(b.leaveUserRequests)
		close(b.sendChatChan)
		close(b.stop)
		close(b.noNameConnChan)
	}()
}

func (b *Broker) broadcast() {
	keys := make([]string, 0, len(b.users))
	for k := range b.users {
		keys = append(keys, k)
	}
	message := MessageToClient{
		Type:  BROADCAST_TYPE,
		Users: keys,
	}
	for _, user := range b.users {
		user.sendMessageChan <- message
	}
}

func (b *Broker) Stop() {
	b.stop <- struct{}{}
}

var wsUpgrader = websocket.Upgrader{}

func (b *Broker) ProcessWsConnect(w http.ResponseWriter, r *http.Request) {
	log.Print("req connection")
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("err upgrade:", err)
		return
	}
	log.Print("req connection waiting for username req")
	go processNoNameConn(conn, b.joinUserRequests)
}

func processNoNameConn(conn *websocket.Conn, joinUserRequests chan<- JoinUserRequest) {
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
		joinUserRequests <- JoinUserRequest{
			UserName: userNameOrNil.UserName,
			conn:     conn,
		}
	}
}
