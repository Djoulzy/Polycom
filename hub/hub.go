package hub

import (
	"encoding/json"
	"fmt"
	"time"

	// "github.com/davecgh/go-spew/spew"
	"github.com/Djoulzy/Tools/clog"
)

const (
	ClientUndefined = 0
	ClientUser      = 1
	ClientServer    = 2
	ClientMonitor   = 3
	Everybody       = 4
	timeStep        = 100 * time.Millisecond // Actualisation 10 par seconde
)

var CTYpeName = [4]string{"Incomming", "Users", "Servers", "Monitors"}

// const (
// 	ReadOnly  = 1
// 	WriteOnly = 2
// 	ReadWrite = 3
// )

type CallToAction func(*Client, []byte)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ID           string
	Send         chan []byte
	Enqueue      chan []byte
	Quit         chan bool
	CallToAction CallToAction

	Addr       string
	CType      int
	Name       string
	Content_id int
	Front_id   string
	App_id     string
	Country    string
	User_agent string
}

type Message struct {
	Level    int
	From     *Client
	UserType int
	Dest     *Client
	Content  []byte
}

var BroadcastQueue [][]byte

type ConnModifier struct {
	Client  *Client
	NewName string
	NewType int
}

type Hub struct {
	// Registered clients.
	Incomming map[string]*Client
	Users     map[string]*Client
	Servers   map[string]*Client
	Monitors  map[string]*Client

	SentMessByTicks int

	FullUsersList [4](map[string]*Client)

	// Inbound messages from the clients.
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
	// Status     chan *Message
	Unicast chan *Message
	Action  chan *Message
	Done    chan bool
}

func NewHub() *Hub {
	hub := &Hub{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),

		Broadcast: make(chan *Message),
		// Status:    make(chan *Message),
		Unicast: make(chan *Message),
		Action:  make(chan *Message),
		Done:    make(chan bool),

		Users:     make(map[string]*Client),
		Incomming: make(map[string]*Client),
		Servers:   make(map[string]*Client),
		Monitors:  make(map[string]*Client),
	}
	hub.FullUsersList = [4](map[string]*Client){hub.Incomming, hub.Users, hub.Servers, hub.Monitors}
	return hub
}

func NewMessage(from *Client, userType int, c *Client, content []byte) *Message {
	m := &Message{
		Level:    1,
		From:     from,
		UserType: userType,
		Dest:     c,
		Content:  content,
	}
	return m
}

func (h *Hub) GetClientByName(name string, userType int) *Client {
	return h.FullUsersList[userType][name]
}

func (h *Hub) UserExists(name string, userType int) bool {
	if h.FullUsersList[userType][name] != nil {
		return true
	} else {
		return false
	}
}

func (h *Hub) IsRegistered(client *Client) bool {
	if h.FullUsersList[client.CType][client.Name] != nil {
		if h.FullUsersList[client.CType][client.Name].ID == client.ID {
			return true
		}
	}
	return false
}

func (h *Hub) register(client *Client) {
	client.ID = fmt.Sprintf("%p", client)

	if h.UserExists(client.Name, client.CType) {
		clog.Warn("Hub", "Register", "Client %s already exists ... replacing", client.Name)
		h.unregister(h.FullUsersList[client.CType][client.Name])
	}

	h.FullUsersList[client.CType][client.Name] = client
	clog.Info("Hub", "Register", "Client %s registered [%s] as %s.", client.Name, client.ID, CTYpeName[client.CType])
}

func (h *Hub) unregister(client *Client) {
	if h.IsRegistered(client) {
		delete(h.FullUsersList[client.CType], client.Name)

		select {
		case client.Quit <- true:
		}

		close(client.Send)
		close(client.Quit)

		if client.CType == ClientServer {
			data := struct {
				SID  string
				DOWN bool
			}{
				client.Name,
				true,
			}
			json, _ := json.Marshal(data)
			mess := NewMessage(client, ClientMonitor, nil, json)
			clog.Trace("Hub", "Unregister", "Broadcasting close of server %s : %s", client.Name, json)
			h.broadcast(mess)
		}
		clog.Info("Hub", "Unregister", "Client %s unregistered [%s] from %s.", client.Name, client.ID, CTYpeName[client.CType])
	}
}

func (h *Hub) Newrole(modif *ConnModifier) {
	if h.UserExists(modif.NewName, modif.NewType) {
		clog.Warn("Hub", "Newrole", "Client already exists ... Deleting")
		h.unregister(h.GetClientByName(modif.NewName, modif.NewType))
	}
	delete(h.FullUsersList[modif.Client.CType], modif.Client.Name)
	modif.Client.Name = modif.NewName
	modif.Client.CType = modif.NewType
	h.FullUsersList[modif.NewType][modif.NewName] = modif.Client
}

func (h *Hub) broadcast(message *Message) {
	if message.UserType == ClientUser {
		BroadcastQueue = append(BroadcastQueue, message.Content)
		clog.Info("Hub", "broadcast", "New message queued: (%d) - %s", len(BroadcastQueue), message.Content)
	} else {
		list := h.FullUsersList[message.UserType]
		for _, client := range list {
			client.Send <- message.Content
			h.SentMessByTicks++
		}
	}
}

func (h *Hub) flushBroadcastQueue() {
	list := h.FullUsersList[ClientUser]
	for i, mess := range BroadcastQueue {
		for _, client := range list {
			client.Send <- mess
			h.SentMessByTicks++
		}
		BroadcastQueue[i] = nil
	}
	BroadcastQueue = BroadcastQueue[:0]
	clog.Info("Hub", "flushBroadcastQueue", "Queue flushed")
}

func (h *Hub) unicast(message *Message) {
	message.Dest.Send <- message.Content
	// clog.Debug("Hub", "unicast", "Unicast Message to %s : %s", message.Dest.Name, message.Content)
	h.SentMessByTicks++
}

func (h *Hub) action(message *Message) {
	// clog.Debug("Hub", "action", "Message %s : %s", message.Dest.Name, message.Content)
	message.Dest.CallToAction(message.Dest, message.Content)
}

func (h *Hub) Run() {
	// ticker := time.NewTicker(timeStep)
	// defer func() {
	// 	ticker.Stop()
	// }()

	for {
		select {
		// case <-ticker.C:
		// 	if len(BroadcastQueue) > 0 {
		// 		h.flushBroadcastQueue()
		// 	}
		case client := <-h.Register:
			h.register(client)
			// client.Consistent <- true
		case client := <-h.Unregister:
			h.unregister(client)
			// client.Consistent <- true
		// case message := <-h.Status:
		// 	h.updateStatus(message)
		case message := <-h.Broadcast:
			h.broadcast(message)
		case message := <-h.Unicast:
			go h.unicast(message)
		case message := <-h.Action:
			go h.action(message)
		case <-h.Done:
			return
		}
	}
}
