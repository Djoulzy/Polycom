package scaling

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	clog "github.com/Djoulzy/Polycom/CLog"

	// "github.com/davecgh/go-spew/spew"

	"github.com/Djoulzy/Polycom/Hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/nettools/TCPServer"
)

var serverCheckPeriod = 10 * time.Second

type NearbyServer struct {
	manager   *TCPServer.Manager
	hubclient *Hub.Client
	connected bool
	cpuload   int
	freeslots int
	httpaddr  string
}

type ServersList struct {
	nodes           map[string]*NearbyServer
	localName       string
	localAddr       string
	MaxServersConns int
	Hub             *Hub.Hub
}

func (slist *ServersList) updateMetrics(serv *NearbyServer, message []byte) {
	hub := serv.manager.Hub
	if len(hub.Monitors)+len(hub.Servers) > 0 {
		clog.Debug("Scaling", "updateMetrics", "Update Metrics for %s", serv.hubclient.Name)

		var metrics monitoring.ServerMetrics

		err := json.Unmarshal(message, &metrics)
		if err != nil {
			clog.Error("Scaling", "updateMetrics", "Cannot reading distant server metrics")
			return
		}
		serv.cpuload = metrics.LAVG
		serv.freeslots = (metrics.MXU - metrics.NBU)
		serv.httpaddr = metrics.HTTPADDR

		slist.AddNewUnknownServer(&metrics.BRTHLST)

		mess := Hub.NewMessage(Hub.ClientMonitor, nil, message)
		hub.Status <- mess
	}
}

func (slist *ServersList) HandShakeTCP(c *Hub.Client, cmd []string) {
	var ctype int

	name := cmd[1]
	addr := cmd[3]
	ctype = Hub.ClientServer
	if len(cmd) != 4 {
		clog.Warn("Scaling", "HandShakeTCP", "Bad connect string from %s, disconnecting.", c.Name)
		c.Hub.Unregister <- c
		return
	}

	if _, ok := c.Hub.Incomming[c.Name]; ok {
		clog.Info("Scaling", "HandShakeTCP", "Identifying %s as %s", c.Name, name)
		c.Hub.Newrole(&Hub.ConnModifier{Client: c, NewName: name, NewType: ctype})
		c.Name = name
		c.Addr = addr
		slist.nodes[addr].hubclient = c

	} else {
		clog.Warn("Scaling", "HandShakeTCP", "Can't identify client... Disconnecting %s.", c.Name)
		c.Hub.Unregister <- c
	}

}

func (slist *ServersList) CallToActionTCP(c *Hub.Client, message []byte) {
	cmd_group := strings.Split(string(message), "|")
	if len(cmd_group) < 2 {
		clog.Warn("Scaling", "CallToActionTCP", "Bad Command '%s', disconnecting client %s.", cmd_group[0], c.Name)
		c.Hub.Unregister <- c
	} else {
		switch cmd_group[0] {
		case "HELLO":
			slist.HandShakeTCP(c, cmd_group)
		case "CMD":
			if c.Identified {
				switch cmd_group[1] {
				case "QUIT":
					clog.Info("Scaling", "CallToActionTCP", "Client %s deconnected normaly.", c.Name)
					c.Hub.Unregister <- c
				case "KILLUSER":
					if c.Hub.UserExists(cmd_group[2], Hub.ClientUser) {
						clog.Info("Scaling", "CallToActionTCP", "Killing user %s", cmd_group[2])

						c.Hub.Unregister <- c.Hub.Users[cmd_group[2]]
					}
				default:
					clog.Warn("Scaling", "CallToActionTCP", "Unknown param %s for command %s", cmd_group[0], cmd_group[1])
					mess := Hub.NewMessage(c.CType, c, []byte(fmt.Sprintf("%s:?", cmd_group[0])))
					c.Hub.Unicast <- mess
				}
			} else {
				mess := Hub.NewMessage(c.CType, c, []byte("HELLO:?"))
				c.Hub.Unicast <- mess
			}
		case "MON":
			clog.Debug("Scaling", "CallToActionTCP", "Metrics received from %s (%s)", c.Name, c.Addr)
			slist.updateMetrics(slist.nodes[c.Addr], message[4:len(message)])
		default:
			clog.Warn("Scaling", "CallToActionTCP", "Unknown Command: %s", cmd_group[0])
		}
	}
}

func (slist *ServersList) checkingNewServers() {
	var wg sync.WaitGroup

	// spew.Dump(slist)
	for addr, node := range slist.nodes {
		if node.hubclient == nil || node.hubclient.Hub == nil {
			conn, err := node.manager.Connect()
			if err == nil {
				clog.Trace("Scaling", "checkingNewServers", "Trying new server -> %s (%s)", node.manager.ServerName, addr)
				wg.Add(1)
				go node.manager.NewOutgoingConn(conn, node.manager.ServerName, slist.localName, slist.localAddr, slist.CallToActionTCP, &wg)
				wg.Wait()
				node.connected = true
				// slist.HandShakeTCP(node.manager.Hub.Incomming[name], []byte(name))
			}
		}
	}
}

func (slist *ServersList) AddNewUnknownServer(list *map[string]bool) {
	for addr, _ := range *list {
		if addr != slist.localAddr {
			clog.Info("Scaling", "AddNewUnknownServer", "Adding %s to scaling procedure.", addr)
			slist.nodes[addr] = &NearbyServer{
				manager: &TCPServer.Manager{
					ServerName: "Unknown",
					Tcpaddr:    addr,
					Hub:        slist.Hub,
				},
				connected: false,
			}
			monitoring.AddBrother <- addr
		}
	}
}

func (slist *ServersList) AddNewConnectedServer(c *Hub.Client) {
	clog.Info("Scaling", "AddNewConnectedServer", "Commit of server %s to scaling procedure.", c.Name)

	c.CallToAction = slist.CallToActionTCP
	slist.nodes[c.Addr] = &NearbyServer{
		manager: &TCPServer.Manager{
			ServerName: c.Name,
			Hub:        c.Hub,
			Tcpaddr:    c.Addr,
		},
		connected: true,
		hubclient: c,
	}
	monitoring.AddBrother <- c.Addr
}

func Init(conf *TCPServer.Manager, list *map[string]string) *ServersList {
	slist := &ServersList{
		nodes:           make(map[string]*NearbyServer),
		localName:       conf.ServerName,
		localAddr:       conf.Tcpaddr,
		MaxServersConns: conf.MaxServersConns,
		Hub:             conf.Hub,
	}

	for name, serv := range *list {
		slist.nodes[serv] = &NearbyServer{
			manager: &TCPServer.Manager{
				ServerName: name,
				Tcpaddr:    serv,
				Hub:        conf.Hub,
			},
			connected: false,
		}
		monitoring.AddBrother <- serv
	}
	return slist
}

func (slist *ServersList) RedirectConnection(client *Hub.Client) bool {
	for _, node := range slist.nodes {
		if node.connected {
			clog.Trace("Scaling", "RedirectConnection", "Server %s CPU: %d Slots: %d", node.hubclient.Name, node.cpuload, node.freeslots)
			if node.cpuload < 80 && node.freeslots > 5 {
				redirect := fmt.Sprintf("REDIRECT|%s", node.httpaddr)
				client.Send <- []byte(redirect)
				clog.Info("Scaling", "RedirectConnection", "Client redirect -> %s (%s)", node.hubclient.Name, node.httpaddr)
				return true
			} else {
				clog.Warn("Scaling", "RedirectConnection", "Server %s full ...", node.hubclient.Name)
			}
		}
	}
	return false
}

func (slist *ServersList) DispatchNewConnection(h *Hub.Hub, name string) {
	message := []byte(fmt.Sprintf("CMD|KILLUSER|%s", name))
	mess := Hub.NewMessage(Hub.ClientServer, nil, message)
	h.Broadcast <- mess
}

func (slist *ServersList) Start() {
	ticker := time.NewTicker(serverCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			slist.checkingNewServers()
		}
	}
}
