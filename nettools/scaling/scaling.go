package scaling

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Djoulzy/Polycom/clog"
	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/nettools/tcpserver"
)

var serverCheckPeriod = 10 * time.Second

type NearbyServer struct {
	manager   *tcpserver.Manager
	hubclient *hub.Client
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
	Hub             *hub.Hub
}

func (slist *ServersList) UpdateMetrics(addr string, message []byte) {
	serv := slist.nodes[addr]
	h := slist.Hub
	if len(h.Monitors)+len(h.Servers) > 0 {
		clog.Debug("Scaling", "updateMetrics", "Update Metrics for %s", serv.manager.Tcpaddr)

		var metrics monitoring.ServerMetrics

		err := json.Unmarshal(message, &metrics)
		if err != nil {
			clog.Error("Scaling", "updateMetrics", "Cannot reading distant server metrics")
			return
		}
		serv.cpuload = metrics.LAVG
		serv.freeslots = (metrics.MXU - metrics.NBU)
		serv.httpaddr = metrics.HTTPADDR

		for name, infos := range metrics.BRTHLST {
			slist.AddNewPotentialServer(name, infos.Tcpaddr)
		}

		if len(h.Monitors) > 0 {
			newSrv := make(map[string]monitoring.Brother)
			newSrv[metrics.SID] = monitoring.Brother{
				Httpaddr: metrics.HTTPADDR,
				Tcpaddr:  metrics.TCPADDR,
			}
			monitoring.AddBrother <- newSrv

			mess := hub.NewMessage(hub.ClientMonitor, nil, message)
			h.Broadcast <- mess
		}
	}
}

// func (slist *ServersList) HandShakeTCP(c *hub.Client, cmd []string) {
// 	var ctype int
//
// 	name := cmd[1]
// 	addr := cmd[3]
// 	ctype = hub.ClientServer
// 	if len(cmd) != 4 {
// 		clog.Warn("Scaling", "HandShakeTCP", "Bad connect string from %s, disconnecting.", c.Name)
// 		c.Hub.Unregister <- c
// 		<-c.Consistent
// 		return
// 	}
//
// 	if _, ok := c.Hub.Incomming[c.Name]; ok {
// 		clog.Info("Scaling", "HandShakeTCP", "Identifying %s as %s", c.Name, name)
// 		c.Hub.Newrole(&hub.ConnModifier{Client: c, NewName: name, NewType: ctype})
// 		c.Name = name
// 		c.Addr = addr
// 		slist.nodes[addr].hubclient = c
//
// 	} else {
// 		clog.Warn("Scaling", "HandShakeTCP", "Can't identify client... Disconnecting %s.", c.Name)
// 		c.Hub.Unregister <- c
// 		<-c.Consistent
// 	}
//
// }

// func (slist *ServersList) CallToActionTCP(c *hub.Client, message []byte) {
// 	cmd_group := strings.Split(string(message), "|")
// 	if len(cmd_group) < 2 {
// 		clog.Warn("Scaling", "CallToActionTCP", "Bad Command '%s', disconnecting client %s.", cmd_group[0], c.Name)
// 		c.Hub.Unregister <- c
// 		<-c.Consistent
// 	} else {
// 		switch cmd_group[0] {
// 		case "HELLO":
// 			slist.HandShakeTCP(c, cmd_group)
// 		case "CMD":
// 			if c.CType != hub.ClientUndefined {
// 				switch cmd_group[1] {
// 				case "QUIT":
// 					clog.Info("Scaling", "CallToActionTCP", "Client %s deconnected normaly.", c.Name)
// 					c.Hub.Unregister <- c
// 					<-c.Consistent
// 				case "KILLUSER":
// 					if c.Hub.UserExists(cmd_group[2], hub.ClientUser) {
// 						clog.Info("Scaling", "CallToActionTCP", "Killing user %s", cmd_group[2])
// 						c.Hub.Unregister <- c.Hub.Users[cmd_group[2]]
// 						<-c.Hub.Users[cmd_group[2]].Consistent
// 					}
// 				default:
// 					clog.Warn("Scaling", "CallToActionTCP", "Unknown param %s for command %s", cmd_group[0], cmd_group[1])
// 					mess := hub.NewMessage(c.CType, c, []byte(fmt.Sprintf("%s:?", cmd_group[0])))
// 					c.Hub.Unicast <- mess
// 				}
// 			} else {
// 				mess := hub.NewMessage(c.CType, c, []byte("HELLO:?"))
// 				c.Hub.Unicast <- mess
// 			}
// 		case "MON":
// 			clog.Debug("Scaling", "CallToActionTCP", "Metrics received from %s (%s)", c.Name, c.Addr)
// 			slist.UpdateMetrics(c.Addr, message[4:len(message)])
// 		default:
// 			clog.Warn("Scaling", "CallToActionTCP", "Unknown Command: %s", cmd_group[0])
// 		}
// 	}
// }

func (slist *ServersList) checkingNewServers() {
	var wg sync.WaitGroup

	// spew.Dump(slist)
	for addr, node := range slist.nodes {
		if node.hubclient == nil || node.hubclient.Hub == nil {
			conn, err := node.manager.Connect()
			if err == nil {
				clog.Trace("Scaling", "checkingNewServers", "Trying new server -> %s (%s)", node.manager.ServerName, addr)
				wg.Add(1)
				go node.manager.NewOutgoingConn(conn, node.manager.ServerName, slist.localName, slist.localAddr, &wg)
				wg.Wait()
				node.connected = true
			}
		}
	}
}

func (slist *ServersList) AddNewConnectedServer(c *hub.Client) {
	clog.Info("Scaling", "AddNewConnectedServer", "Commit of server %s to scaling procedure.", c.Name)

	slist.nodes[c.Addr] = &NearbyServer{
		manager: &tcpserver.Manager{
			ServerName: c.Name,
			Hub:        c.Hub,
			Tcpaddr:    c.Addr,
		},
		connected: true,
		hubclient: c,
	}
}

func (slist *ServersList) AddNewPotentialServer(name string, addr string) {
	if slist.nodes[addr] == nil {
		if addr != slist.localAddr {
			slist.nodes[addr] = &NearbyServer{
				manager: &tcpserver.Manager{
					ServerName: name,
					Tcpaddr:    addr,
					Hub:        slist.Hub,
				},
				connected: false,
			}
		}
	}
}

func Init(conf *tcpserver.Manager, list *map[string]string) *ServersList {
	slist := &ServersList{
		nodes:           make(map[string]*NearbyServer),
		localName:       conf.ServerName,
		localAddr:       conf.Tcpaddr,
		MaxServersConns: conf.MaxServersConns,
		Hub:             conf.Hub,
	}

	if list != nil {
		for name, serv := range *list {
			slist.AddNewPotentialServer(name, serv)
		}
	}
	return slist
}

func (slist *ServersList) RedirectConnection(client *hub.Client) bool {
	for _, node := range slist.nodes {
		if node.connected {
			clog.Trace("Scaling", "RedirectConnection", "Server %s CPU: %d Slots: %d", node.hubclient.Name, node.cpuload, node.freeslots)
			if node.cpuload < 80 && node.freeslots > 5 {
				redirect := fmt.Sprintf("[RDCT]%s", node.httpaddr)
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

func (slist *ServersList) DispatchNewConnection(h *hub.Hub, name string) {
	message := []byte(fmt.Sprintf("[KILL]%s", name))
	mess := hub.NewMessage(hub.ClientServer, nil, message)
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
