package scaling

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/nettools/tcpserver"
	"github.com/Djoulzy/Tools/clog"
)

var serverCheckPeriod = 10 * time.Second

type NearbyServer struct {
	hubclient   *hub.Client
	distantName string
	connected   bool
	cpuload     int
	freeslots   int
	httpaddr    string
	tcpaddr     string
}

type ServersList struct {
	nodes           map[string]*NearbyServer
	tcpmanager      *tcpserver.Manager
	localName       string
	localAddr       string
	MaxServersConns int
	Hub             *hub.Hub
}

func (slist *ServersList) UpdateMetrics(addr string, message []byte) {
	serv := slist.nodes[addr]
	h := slist.Hub
	if len(h.Monitors)+len(h.Servers) > 0 {
		clog.Debug("Scaling", "updateMetrics", "Update Metrics for %s", serv.tcpaddr)

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

		newSrv := make(map[string]monitoring.Brother)
		newSrv[metrics.SID] = monitoring.Brother{
			Httpaddr: metrics.HTTPADDR,
			Tcpaddr:  metrics.TCPADDR,
		}
		monitoring.AddBrother <- newSrv

		if len(h.Monitors) > 0 {
			mess := hub.NewMessage(nil, hub.ClientMonitor, nil, message)
			h.Broadcast <- mess
		}
	}
}

func (slist *ServersList) checkingNewServers() {
	// var wg sync.WaitGroup

	// spew.Dump(slist)
	// for addr, node := range slist.nodes {
	// 	if node.hubclient == nil || node.hubclient.Hub == nil {
	// 		conn, err := slist.tcpmanager.Connect(addr)
	// 		if err == nil {
	// 			clog.Trace("Scaling", "checkingNewServers", "Trying new server -> %s (%s)", node.distantName, addr)
	// 			wg.Add(1)
	// 			go slist.tcpmanager.NewOutgoingConn(conn, node.distantName, &wg)
	// 			wg.Wait()
	// 			node.connected = true
	// 		}
	// 	}
	// }
}

func (slist *ServersList) AddNewConnectedServer(c *hub.Client) {
	clog.Info("Scaling", "AddNewConnectedServer", "Commit of server %s to scaling procedure.", c.Name)
	slist.nodes[c.Addr] = &NearbyServer{
		// manager: &tcpserver.Manager{
		// 	ServerName: c.Name,
		// 	Hub:        c.Hub,
		// 	Tcpaddr:    c.Addr,
		// },
		distantName: c.Name,
		tcpaddr:     c.Addr,
		connected:   true,
		hubclient:   c,
	}
}

func (slist *ServersList) AddNewPotentialServer(name string, addr string) {
	if slist.nodes[addr] == nil {
		if addr != slist.localAddr {
			clog.Info("Scaling", "AddNewPotentialServer", "New server : %s (%s)", name, addr)
			slist.nodes[addr] = &NearbyServer{
				// manager: &tcpserver.Manager{
				// 	ServerName: name,
				// 	Tcpaddr:    addr,
				// 	Hub:        slist.Hub,
				// },

				distantName: name,
				tcpaddr:     addr,
				connected:   false,
			}
		}
	}
}

func Init(conf *tcpserver.Manager, list *map[string]string) *ServersList {
	slist := &ServersList{
		nodes:           make(map[string]*NearbyServer),
		tcpmanager:      conf,
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
			if node.cpuload < 80 && node.freeslots > 0 {
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
	mess := hub.NewMessage(nil, hub.ClientServer, nil, message)
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
