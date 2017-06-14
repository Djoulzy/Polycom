package tcpserver

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	// "github.com/davecgh/go-spew/spew"
	"github.com/Djoulzy/Polycom/clog"
	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/urlcrypt"
)

var (
	Newline = []byte{'\r', '\n'}
	Space   = []byte{' '}
)

type Manager struct {
	Tcpaddr                  string
	ServerName               string
	Hub                      *hub.Hub
	MaxServersConns          int
	ConnectTimeOut           int
	WriteTimeOut             int
	ScalingCheckServerPeriod int
	CallToAction             func(*hub.Client, []byte)
	Cryptor                  *urlcrypt.Cypher
}

func (m *Manager) reader(c *hub.Client) {
	defer func() {
		c.Conn.(*net.TCPConn).Close()
	}()
	conn := c.Conn.(*net.TCPConn)
	// message := make([]byte, 1024)
	for {
		// conn.SetReadDeadline(time.Now().Add(time.Second * 10))
		message, err := bufio.NewReader(conn).ReadBytes('\n')
		if err != nil {
			clog.Trace("TCPserver", "reader", "closing conn %s", err)
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, Newline, Space, -1))
		// clog.Trace("TCPserver", "reader", "Reading %s", message)
		// long, err := conn.Read(message)
		// if err != nil {
		// 	break
		// }
		// message = message[:long-1]
		// spew.Dump(message)
		go m.CallToAction(c, message)
	}
}

func (m *Manager) writer(c *hub.Client) {
	defer func() {
		c.Conn.(*net.TCPConn).Close()
	}()

	conn := c.Conn.(*net.TCPConn)
	for {
		select {
		case <-c.Quit:
			clog.Trace("TCPserver", "writer", "closing conn")
			return
		case message, ok := <-c.Send:
			// clog.Debug("TCPserver", "writer", "Sending %s", message)
			if !ok {
				// The hub closed the channel.
				return
			}

			err := conn.SetWriteDeadline(time.Now().Add(time.Second))
			if err != nil {

				return
			}
			message = append(message, Newline...)
			conn.Write(message)
		}
	}
}

func GetAddr(c *hub.Client) string {
	addr := c.Conn.(*net.TCPConn).RemoteAddr().String()
	ip := strings.Split(string(addr), "|")
	return ip[0]
}

func (m *Manager) Connect() (*net.TCPConn, error) {
	conn, err := net.DialTimeout("tcp", m.Tcpaddr, time.Second*time.Duration(m.ConnectTimeOut))
	// addr, _ := net.ResolveTCPAddr("tcp", m.Tcpaddr)
	// conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		clog.Error("TCPserver", "Connect", "Can't connect to server %s", m.Tcpaddr)
		return nil, err
	}
	return conn.(*net.TCPConn), err
}

func (m *Manager) newClient(conn *net.TCPConn, name string) *hub.Client {
	client := &hub.Client{Hub: m.Hub, Conn: conn, Consistent: make(chan bool), Quit: make(chan bool),
		CType: hub.ClientUndefined, Send: make(chan []byte, 256), CallToAction: m.CallToAction, Addr: conn.RemoteAddr().String(),
		Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: "TCP Socket"}
	m.Hub.Register <- client
	<-client.Consistent
	return client
}

func (m *Manager) NewOutgoingConn(conn *net.TCPConn, toName string, fromName string, fromAddr string, wg *sync.WaitGroup) {
	clog.Debug("TCPserver", "NewOutgoingConn", "Contacting %s", conn.RemoteAddr().String())
	client := m.newClient(conn, toName)
	mess := hub.NewMessage(client.CType, client, []byte(fmt.Sprintf("HELLO|%s|LISTN|%s", fromName, fromAddr)))
	m.Hub.Unicast <- mess

	go m.writer(client)
	(*wg).Done()
	m.reader(client)
	m.Hub.Unregister <- client
	<-client.Consistent
}

func (m *Manager) NewIncommingConn(conn *net.TCPConn, wg *sync.WaitGroup) {
	client := m.newClient(conn, conn.RemoteAddr().String())
	mess := hub.NewMessage(client.CType, client, []byte(fmt.Sprintf("HELLO|%s|LISTN|%s", m.ServerName, m.Tcpaddr)))
	m.Hub.Unicast <- mess

	go m.writer(client)
	(*wg).Done()
	m.reader(client)
	m.Hub.Unregister <- client
	<-client.Consistent
}

func (m *Manager) Start(conf *Manager) {
	var wg sync.WaitGroup

	m = conf

	formatedaddr, _ := net.ResolveTCPAddr("tcp", m.Tcpaddr)
	ln, err := net.ListenTCP("tcp", formatedaddr)
	if err != nil {
		clog.Error("TCPserver", "Start", "%s", err)
	}

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			// handle error
		}
		wg.Add(1)
		go m.NewIncommingConn(conn, &wg)
		wg.Wait()
	}
}
