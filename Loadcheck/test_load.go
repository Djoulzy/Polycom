// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Djoulzy/Polycom/CLog"
	"github.com/Djoulzy/Polycom/URLCrypt"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 5 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
}

var cryptor = &urlcrypt.Cypher{
	HASH_SIZE: 8,
	HEX_KEY:   []byte("d87fbb277eefe245ee384b6098637513462f5151336f345778706b462f724473"),
	HEX_IV:    []byte("046b51957f00c25929e8ccaad3bfe1a7"),
}

var httpaddr = flag.String("httpaddr", "localhost:8080", "http service address")
var tcpaddr = flag.String("tcpaddr", "localhost:8081", "tcp service address")
var mu sync.Mutex
var wu sync.Mutex
var wg sync.WaitGroup

// Conn is an middleman between the websocket connection and the hub.
type Conn struct {
	name int
	ws   *websocket.Conn
	send chan []byte
}

type ClientsList map[int]*Conn

var Clients = make(ClientsList)

func TryRedirect(c *Conn, addr string) {
	close(c.send)
	mu.Lock()
	Clients[c.name] = nil
	mu.Unlock()
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	log.Printf("Try redirect: %s", addr)
	wg.Add(1)
	go connect(c.name, u)
	wg.Wait()
	connString, _ := cryptor.Encrypt_b64(fmt.Sprintf("LOAD_%d|253907|WEB|wmsa_BR|BR|iPhone", c.name))
	Clients[c.name].send <- []byte(connString)
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Conn) readPump() {
	defer func() {
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		cmd_group := strings.Split(string(message), "|")
		if cmd_group[0] == "REDIRECT" {
			go TryRedirect(c, cmd_group[1])
			break
		}
	}
}

// write writes a message with the given message type and payload.
func (c *Conn) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				cm := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Disconnected")
				if err := c.write(websocket.CloseMessage, cm); err != nil {
				}
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func connect(i int, u url.URL) {
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	conn := &Conn{name: i, send: make(chan []byte, 256), ws: ws}
	mu.Lock()
	Clients[i] = conn
	mu.Unlock()
	wg.Done()

	// log.Printf("Conn: %s\n", Clients[i])
	go conn.writePump()
	conn.readPump()
	// <-readeyWrite
	// log.Printf("HTTPServer: connecting to %s", u.String())
}

func main() {
	flag.Parse()

	clog.LogLevel = 5
	clog.StartLogging = true

	u := url.URL{Scheme: "ws", Host: *httpaddr, Path: "/ws"}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go connect(i, u)
		wg.Wait()
		duration := time.Second / 100
		time.Sleep(duration)
		connString, _ := cryptor.Encrypt_b64(fmt.Sprintf("LOAD_%d|253907|WEB|wmsa_BR|BR|iPhone", i))
		clog.Debug("test_load", "main", "Connecting %s ...", connString)
		Clients[i].send <- []byte(connString)
	}

	// duration := time.Second
	// time.Sleep(duration)

	// for index, client := range Clients {
	// 	connString := fmt.Sprintf("LOAD_%d,253907,WEB,wmsa,BR", index)
	// 	client.send <- []byte(connString)

	// 	duration := time.Second / 10
	// 	time.Sleep(duration)
	// }

	for {
		// mu.Lock()
		for index, client := range Clients {
			connString := fmt.Sprintf("LOAD_%d", index)
			if client.ws != nil {
				client.send <- []byte(connString)
				duration := time.Second / 10
				time.Sleep(duration)
			} else {
				delete(Clients, index)
			}
		}
		// mu.Unlock()
	}
}
