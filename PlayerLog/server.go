package main

import (
	"fmt"
	"strconv"
	"strings"

	clog "github.com/Djoulzy/Polycom/CLog"
	urlcrypt "github.com/Djoulzy/Polycom/URLCrypt"
	scaling "github.com/Djoulzy/Polycom/nettools/Scaling"

	"github.com/Djoulzy/Polycom/Config"
	"github.com/Djoulzy/Polycom/Hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/nettools/HTTPServer"
	"github.com/Djoulzy/Polycom/nettools/TCPServer"
)

var conf *Config.Data

var cryptor *urlcrypt.Cypher

var HTTPManager HTTPServer.Manager
var TCPManager TCPServer.Manager
var scaleList *scaling.ServersList

func HandShakeHTTP(c *Hub.Client, message []byte) {
	hub := c.Hub

	if string(message) == "Status:General" {
		clog.Info("server", "HandShakeHTTP", "New Status Client %s", c.Name)
		if len(hub.Monitors) >= conf.MaxMonitorsConns {
			hub.Unregister <- c
			<-c.Consistent
		} else {
			hub.Newrole(&Hub.ConnModifier{Client: c, NewName: c.Name, NewType: Hub.ClientMonitor})
			// c.Mode = Hub.WriteOnly
			// test := <-hub.Done
			// log.Printf("Chann %s\n", test)
		}
	} else {
		uncrypted_message, _ := cryptor.Decrypt_b64(string(message))
		clog.Info("server", "HandShakeHTTP", "New User Client %s (%s)", c.Name, uncrypted_message)
		infos := strings.Split(string(uncrypted_message), "|")
		if len(infos) != 6 {
			clog.Warn("server", "HandShakeHTTP", "Bad Handshake format ... Disconnecting")
			hub.Unregister <- c
			<-c.Consistent
			return
		}
		content_id, err := strconv.Atoi(strings.TrimSpace(infos[1]))
		if err != nil {
			clog.Warn("server", "HandShakeHTTP", "Unrecognized content_id ... Disconnecting")
			hub.Unregister <- c
			<-c.Consistent
			return
		}

		actualName := c.Name
		newName := infos[0]

		// lock.Lock()
		if hub.UserExists(actualName, Hub.ClientUndefined) {
			if len(hub.Users) >= conf.MaxUsersConns && !hub.UserExists(newName, Hub.ClientUser) {
				clog.Warn("server", "HandShakeHTTP", "Too many Users connections, rejecting %s (In:%d/Cl:%d).", actualName, len(hub.Incomming), len(hub.Users))
				if !scaleList.RedirectConnection(c) {
					clog.Error("server", "HandShakeHTTP", "NO FREE SLOTS !!!")
				}
				hub.Unregister <- c
				<-c.Consistent
			} else {
				c.Hub.Newrole(&Hub.ConnModifier{Client: c, NewName: newName, NewType: Hub.ClientUser})
				// c.Mode = Hub.ReadWrite
				c.Content_id = content_id
				c.Front_id = strings.TrimSpace(infos[2])
				c.App_id = strings.TrimSpace(infos[3])
				c.Country = strings.TrimSpace(infos[4])
				clog.Info("server", "HandShakeHTTP", "Identifying %s as %s", actualName, newName)

				scaleList.DispatchNewConnection(hub, c.Name)
				// <-hub.Done
			}
		} else {
			clog.Warn("server", "HandShakeHTTP", "Can't identify client... Disconnecting %s.", c.Name)
			hub.Unregister <- c
			<-c.Consistent
		}
		// lock.Unlock()
	}
}

func CallToActionHTTP(c *Hub.Client, message []byte) {
	if c.CType != Hub.ClientUndefined {
		if c.CType == Hub.ClientUser {
			mess := Hub.NewMessage(Hub.ClientUser, nil, message)
			c.Hub.Broadcast <- mess
		} else {
		}
	} else {
		HandShakeHTTP(c, message)
	}
}

func HandShakeTCP(c *Hub.Client, cmd []string) {
	var ctype int

	name := cmd[1]
	if len(cmd) > 2 {
		ctype = Hub.ClientServer
		if len(c.Hub.Servers) >= conf.MaxServersConns {
			clog.Warn("server", "HandShakeTCP", "Too many Server connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(c.Hub.Incomming), len(c.Hub.Servers))
			c.Hub.Unregister <- c
			<-c.Consistent
			return
		}
	} else {
		ctype = Hub.ClientUser
		if len(c.Hub.Users) >= conf.MaxUsersConns {
			clog.Warn("server", "HandShakeTCP", "Too many Users connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(c.Hub.Incomming), len(c.Hub.Users))
			c.Hub.Unregister <- c
			<-c.Consistent
			return
		}
	}

	if _, ok := c.Hub.Incomming[c.Name]; ok {
		clog.Info("server", "HandShakeTCP", "Identifying %s as %s", c.Name, name)
		c.Hub.Newrole(&Hub.ConnModifier{Client: c, NewName: name, NewType: ctype})
		if len(cmd) == 4 {
			c.Addr = cmd[3]
			scaleList.AddNewConnectedServer(c)
		}
	} else {
		clog.Warn("server", "HandShakeTCP", "Can't identify client... Disconnecting %s.", c.Name)
		c.Hub.Unregister <- c
		<-c.Consistent
	}
}

func CallToActionTCP(c *Hub.Client, message []byte) {
	cmd_group := strings.Split(string(message), "|")
	if len(cmd_group) < 2 {
		clog.Warn("server", "CallToActionTCP", "Bad Command '%s', disconnecting client %s.", cmd_group[0], c.Name)
		c.Hub.Unregister <- c
		<-c.Consistent
	} else {
		switch cmd_group[0] {
		case "HELLO":
			HandShakeTCP(c, cmd_group)
		case "CMD":
			if c.CType != Hub.ClientUndefined {
				switch cmd_group[1] {
				case "quit":
					clog.Info("server", "CallToActionTCP", "Client %s deconnected normaly.", c.Name)
					c.Hub.Unregister <- c
					<-c.Consistent
				default:
					clog.Warn("server", "CallToActionTCP", "Unknown param %s for command %s", cmd_group[0], cmd_group[1])
					mess := Hub.NewMessage(c.CType, c, []byte(fmt.Sprintf("%s:?", cmd_group[0])))
					c.Hub.Unicast <- mess
				}
			} else {
				mess := Hub.NewMessage(c.CType, c, []byte("HELLO|?"))
				c.Hub.Unicast <- mess
			}
		case "MON":
			clog.Warn("server", "CallToActionTCP", "Wrong TCPManager: %s", cmd_group[0])
		default:
			clog.Warn("server", "CallToActionTCP", "Unknown Command: %s", cmd_group[0])
		}
	}
}

func main() {
	conf, _ = Config.Load()

	clog.LogLevel = conf.LogLevel
	clog.StartLogging = conf.StartLogging

	cryptor = &urlcrypt.Cypher{
		HASH_SIZE: conf.HASH_SIZE,
		HEX_KEY:   []byte(conf.HEX_KEY),
		HEX_IV:    []byte(conf.HEX_IV),
	}

	hub := Hub.NewHub()
	// hub.UsersMonitor = monitoring.UserMonitoring
	clog.Debug("server", "main", "Starting")

	go hub.Run()

	mon_params := &monitoring.Params{
		ServerID:          conf.Name,
		Httpaddr:          conf.HTTPaddr,
		Tcpaddr:           conf.TCPaddr,
		MaxUsersConns:     conf.MaxUsersConns,
		MaxMonitorsConns:  conf.MaxMonitorsConns,
		MaxServersConns:   conf.MaxServersConns,
		MaxIncommingConns: conf.MaxIncommingConns,
	}
	go monitoring.Start(hub, mon_params)

	tcp_params := &TCPServer.Manager{
		ServerName:               conf.Name,
		Tcpaddr:                  conf.TCPaddr,
		Hub:                      hub,
		ConnectTimeOut:           conf.ConnectTimeOut,
		WriteTimeOut:             conf.WriteTimeOut,
		ScalingCheckServerPeriod: conf.ScalingCheckServerPeriod,
		MaxServersConns:          conf.MaxServersConns,
	}

	scaleList = scaling.Init(tcp_params, &conf.KnownBrothers.Servers)
	go scaleList.Start()
	// go scaling.Start(ScalingServers)

	http_params := &HTTPServer.Manager{
		ServerName:       conf.Name,
		Httpaddr:         conf.HTTPaddr,
		Hub:              hub,
		ReadBufferSize:   conf.ReadBufferSize,
		WriteBufferSize:  conf.WriteBufferSize,
		HandshakeTimeout: conf.HandshakeTimeout,
	}
	clog.Info("server", "main", "HTTP Server starting listening on %s", conf.HTTPaddr)
	go HTTPManager.Start(http_params, CallToActionHTTP)

	clog.Info("server", "main", "TCP Server starting listening on %s", conf.TCPaddr)
	TCPManager.Start(tcp_params, CallToActionTCP)
}
