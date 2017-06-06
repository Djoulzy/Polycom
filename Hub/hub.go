package Hub

import (
	"encoding/json"
	"fmt"

	// "github.com/davecgh/go-spew/spew"
	"github.com/Djoulzy/Polycom/CLog"
)

const (
	ClientUndefined = 0
	ClientUser      = 1
	ClientServer    = 2
	ClientMonitor   = 3
	Everybody       = 4
)

var CTYpeName = [4]string{"Incomming", "Users", "Servers", "Monitors"}

const (
	ReadOnly  = 1
	WriteOnly = 2
	ReadWrite = 3
)

type CallToAction func(*Client, []byte)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ID  string
	Hub *Hub

	// WriteProtect sync.Mutex
	// ReadProtect  sync.Mutex
	Conn interface{}

	Send         chan []byte
	Quit         chan bool
	CallToAction CallToAction

	Identified bool
	Addr       string
	CType      int
	Name       string
	Content_id int
	Front_id   string
	App_id     string
	Country    string
	User_agent string
	Mode       int
}

type Message struct {
	UserType int
	Dest     *Client
	Content  []byte
}

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
	Status     chan *Message
	Unicast    chan *Message
	Action     chan *Message
	Done       chan bool
}

func NewHub() *Hub {
	hub := &Hub{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),

		Broadcast: make(chan *Message),
		Status:    make(chan *Message),
		Unicast:   make(chan *Message),
		Action:    make(chan *Message),
		Done:      make(chan bool),

		Users:     make(map[string]*Client),
		Incomming: make(map[string]*Client),
		Servers:   make(map[string]*Client),
		Monitors:  make(map[string]*Client),
	}
	hub.FullUsersList = [4](map[string]*Client){hub.Incomming, hub.Users, hub.Servers, hub.Monitors}
	return hub
}

func NewMessage(userType int, c *Client, content []byte) *Message {
	m := &Message{
		UserType: userType,
		Dest:     c,
		Content:  content,
	}
	return m
}

func (h *Hub) register(client *Client) {
	client.ID = fmt.Sprintf("%p", client)

	if h.UserExists(client.Name, client.CType) {
		h.unregister(h.FullUsersList[client.CType][client.Name])
		clog.Warn("Hub", "Register", "Client %s already exists ... replacing", client.Name)
	}

	h.FullUsersList[client.CType][client.Name] = client
	clog.Info("Hub", "Register", "Client %s registered [%s] as %s.", client.Name, client.ID, CTYpeName[client.CType])
}

func (h *Hub) unregister(client *Client) {
	var list map[string]*Client

	list = h.FullUsersList[client.CType]
	if _, ok := list[client.Name]; ok {
		if client.Conn != nil {
			client.Quit <- true
		}
		client.Hub = nil
		close(list[client.Name].Send)
		delete(list, client.Name)
		clog.Info("Hub", "Unregister", "Client %s unregistered [%s] from %s.", client.Name, client.ID, CTYpeName[client.CType])
		if client.CType == ClientServer {
			data := struct {
				SID  string
				DOWN bool
			}{
				client.Name,
				true,
			}
			json, _ := json.Marshal(data)
			mess := NewMessage(ClientMonitor, nil, json)
			clog.Trace("Hub", "Unregister", "Broadcasting close of server %s : %s", client.Name, json)
			h.updateStatus(mess)
		}
	} else {
		// clog.Error("Hub", "Unregister", "%s", spew.Sdump(client))
	}
}

func (h *Hub) Newrole(modif *ConnModifier) {
	// clog.Test("Hub", "newrole", "%s", modif)
	delete(h.FullUsersList[modif.Client.CType], modif.Client.Name)
	modif.Client.Name = modif.NewName
	modif.Client.CType = modif.NewType
	modif.Client.Identified = true
	h.FullUsersList[modif.NewType][modif.NewName] = modif.Client
}

func (h *Hub) updateStatus(message *Message) {
	var list map[string]*Client

	list = h.FullUsersList[message.UserType]
	for _, client := range list {
		if client.Identified && client.Mode != ReadOnly {
			select {
			case client.Send <- message.Content:
			default:
				h.unregister(client)
			}
		}
	}
}

func (h *Hub) broadcast(message *Message) {
	var list map[string]*Client

	list = h.FullUsersList[message.UserType]
	for _, client := range list {
		if client.Hub != nil {
			client.Send <- message.Content
			clog.Debug("Hub", "broadcast", "Broadcast %s Message : %s", CTYpeName[message.UserType], message.Content)
			h.SentMessByTicks++
		}
	}
}

func (h *Hub) unicast(message *Message) {
	message.Dest.Send <- message.Content
	clog.Debug("Hub", "unicast", "Unicast Message to %s : %s", message.Dest.Name, message.Content)
	h.SentMessByTicks++
}

func (h *Hub) action(message *Message) {
	// clog.Debug("Hub", "action", "Message %s : %s", message.Dest.Name, message.Content)
	message.Dest.CallToAction(message.Dest, message.Content)
}

func (h *Hub) UserExists(name string, ctype int) bool {
	var list map[string]*Client

	switch ctype {
	case ClientUndefined:
		list = h.Incomming
	case ClientUser:
		list = h.Users
	case ClientMonitor:
		list = h.Monitors
	case ClientServer:
		list = h.Servers
	}

	if list[name] != nil {
		return true
	} else {
		return false
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.register(client)
		case client := <-h.Unregister:
			h.unregister(client)
		case message := <-h.Status:
			h.updateStatus(message)
		case message := <-h.Broadcast:
			h.broadcast(message)
		case message := <-h.Unicast:
			go h.unicast(message)
		case message := <-h.Action:
			go h.action(message)
		}
	}
}
