// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/Djoulzy/Polycom/urlcrypt"
	"github.com/Djoulzy/Tools/clog"
	"github.com/Djoulzy/Tools/config"
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

var cryptor *urlcrypt.Cypher

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
	connString, _ := cryptor.Encrypt_b64(fmt.Sprintf("LOAD_%d|wmsa_BR|USER", c.name))
	Clients[c.name].send <- append([]byte("[HELO]"), []byte(connString)...)
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
		cmd_group := string(message[0:6])
		action_group := message[6:]
		if cmd_group == "[RDCT]" {
			go TryRedirect(c, string(action_group))
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
	config.Load("server.ini", conf)

	clog.LogLevel = 5
	clog.StartLogging = true

	u := url.URL{Scheme: "ws", Host: conf.HTTPaddr, Path: "/ws"}

	cryptor = &urlcrypt.Cypher{
		HASH_SIZE: conf.HASH_SIZE,
		HEX_KEY:   []byte(conf.HEX_KEY),
		HEX_IV:    []byte(conf.HEX_IV),
	}

	for i := 0; i < 5000; i++ {
		wg.Add(1)
		go connect(i, u)
		wg.Wait()
		passPhrase := fmt.Sprintf("LOAD_%d|wmsa_BR|USER", i)
		connString, _ := cryptor.Encrypt_b64(passPhrase)
		clog.Debug("test_load", "main", "Connecting %s [%s] ...", passPhrase, connString)
		Clients[i].send <- append([]byte("[HELO]"), []byte(connString)...)
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
				client.send <- append([]byte("[BCST]"), []byte(connString)...)
				duration := time.Second
				time.Sleep(duration)
			} else {
				delete(Clients, index)
			}
		}
		// mu.Unlock()
	}
}
