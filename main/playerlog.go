package main

import (
	"fmt"
	"strings"

	"github.com/Djoulzy/Polycom/clog"
	"github.com/Djoulzy/Polycom/hub"
)

func welcomeNewMonitor(c *hub.Client, newName string, app_id string) {
	h := c.Hub
	if len(h.Monitors) >= conf.MaxMonitorsConns {
		h.Unregister <- c
		<-c.Consistent
	} else {
		h.Newrole(&hub.ConnModifier{Client: c, NewName: c.Name, NewType: hub.ClientMonitor})
		c.App_id = app_id
		clog.Info("server", "welcomeNewMonitor", "Accepting %s", c.Name)
	}
}

func welcomeNewUser(c *hub.Client, newName string, app_id string) {
	h := c.Hub
	if h.UserExists(c.Name, hub.ClientUndefined) {
		if len(h.Users) >= conf.MaxUsersConns && !h.UserExists(newName, hub.ClientUser) {
			clog.Warn("server", "welcomeNewUser", "Too many Users connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(h.Incomming), len(h.Users))
			if !scaleList.RedirectConnection(c) {
				clog.Error("server", "welcomeNewUser", "NO FREE SLOTS !!!")
			}
			h.Unregister <- c
			<-c.Consistent
		} else {
			clog.Info("server", "welcomeNewUser", "Identifying %s as %s", c.Name, newName)
			h.Newrole(&hub.ConnModifier{Client: c, NewName: newName, NewType: hub.ClientUser})
			c.App_id = app_id
			scaleList.DispatchNewConnection(h, c.Name)
		}
	} else {
		clog.Warn("server", "welcomeNewUser", "Can't identify client... Disconnecting %s.", c.Name)
		h.Unregister <- c
		<-c.Consistent
	}
}

func welcomeNewServer(c *hub.Client, newName string, addr string) {
	h := c.Hub
	if len(h.Servers) >= conf.MaxServersConns {
		clog.Warn("server", "welcomeNewServer", "Too many Server connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(h.Incomming), len(h.Servers))
		h.Unregister <- c
		<-c.Consistent
		return
	}

	if h.UserExists(c.Name, hub.ClientUndefined) {
		clog.Info("server", "welcomeNewServer", "Identifying %s as %s", c.Name, newName)
		h.Newrole(&hub.ConnModifier{Client: c, NewName: newName, NewType: hub.ClientServer})
		c.Addr = addr
		scaleList.AddNewConnectedServer(c)
	} else {
		clog.Warn("server", "welcomeNewServer", "Can't identify server... Disconnecting %s.", c.Name)
		h.Unregister <- c
		<-c.Consistent
	}
}

func HandShake(c *hub.Client, message []byte) {
	h := c.Hub
	uncrypted_message, _ := Cryptor.Decrypt_b64(string(message))
	clog.Info("server", "HandShake", "New Incomming Client %s (%s)", c.Name, uncrypted_message)
	infos := strings.Split(string(uncrypted_message), "|")
	if len(infos) != 3 {
		clog.Warn("server", "HandShake", "Bad Handshake format ... Disconnecting")
		h.Unregister <- c
		<-c.Consistent
		return
	}

	App_id := strings.TrimSpace(infos[1])
	newName := strings.TrimSpace(infos[0])
	switch infos[2] {
	case "MNTR":
		welcomeNewMonitor(c, newName, App_id)
	case "SERV":
		welcomeNewServer(c, newName, App_id)
	case "USER":
		welcomeNewUser(c, newName, App_id)
	default:
		h.Unregister <- c
		<-c.Consistent
	}
}

func CallToAction(c *hub.Client, message []byte) {
	h := c.Hub
	cmd_group := string(message[0:6])
	action_group := message[6:]

	if c.CType != hub.ClientUndefined {
		switch cmd_group {
		case "[BCST]":
			mess := hub.NewMessage(hub.ClientUser, nil, action_group)
			h.Broadcast <- mess
		case "[UCST]":
		case "[STOR]":
			Storage.NewRecord(string(action_group))
		case "[QUIT]":
			h.Unregister <- c
			<-c.Consistent
		default:
			mess := hub.NewMessage(c.CType, c, []byte(fmt.Sprintf("%s:?", cmd_group)))
			h.Unicast <- mess
		}
	} else {
		switch cmd_group {
		case "[HELO]":
			// [HELO]<unique_id>|<app_id ou addr_ip>|<client_type>
			HandShake(c, action_group)
		default:
			clog.Warn("server", "CallToAction", "Bad Command '%s', disconnecting client %s.", cmd_group, c.Name)
			h.Unregister <- c
			<-c.Consistent
		}
	}
}

// func HandShakeTCP(c *hub.Client, cmd []byte) {
// 	var ctype int
//
// 	name := cmd[1]
// 	if len(cmd) > 2 {
// 		ctype = hub.ClientServer
// 		if len(c.Hub.Servers) >= conf.MaxServersConns {
// 			clog.Warn("server", "HandShakeTCP", "Too many Server connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(c.Hub.Incomming), len(c.Hub.Servers))
// 			c.Hub.Unregister <- c
// 			<-c.Consistent
// 			return
// 		}
// 	} else {
// 		ctype = hub.ClientUser
// 		if len(c.Hub.Users) >= conf.MaxUsersConns {
// 			clog.Warn("server", "HandShakeTCP", "Too many Users connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(c.Hub.Incomming), len(c.Hub.Users))
// 			c.Hub.Unregister <- c
// 			<-c.Consistent
// 			return
// 		}
// 	}
//
// 	if _, ok := c.Hub.Incomming[c.Name]; ok {
// 		clog.Info("server", "HandShakeTCP", "Identifying %s as %s", c.Name, name)
// 		c.Hub.Newrole(&hub.ConnModifier{Client: c, NewName: name, NewType: ctype})
// 		if len(cmd) == 4 {
// 			c.Addr = cmd[3]
// 			scaleList.AddNewConnectedServer(c)
// 		}
// 	} else {
// 		clog.Warn("server", "HandShakeTCP", "Can't identify client... Disconnecting %s.", c.Name)
// 		c.Hub.Unregister <- c
// 		<-c.Consistent
// 	}
// }

// func CallToActionTCP(c *hub.Client, message []byte) {
// 	cmd_group := string(message[0:6])
// 	action_group := message[6:]
// 	switch cmd_group {
// 	case "[HELO]":
// 		HandShakeTCP(c, action_group)
// 	case "[CMMD]":
// 		if c.CType != hub.ClientUndefined {
// 			switch string(action_group) {
// 			case "quit":
// 				clog.Info("server", "CallToActionTCP", "Client %s deconnected normaly.", c.Name)
// 				c.Hub.Unregister <- c
// 				<-c.Consistent
// 			default:
// 				clog.Warn("server", "CallToActionTCP", "Unknown param %s for command %s", cmd_group[0], cmd_group[1])
// 				mess := hub.NewMessage(c.CType, c, []byte(fmt.Sprintf("%s:?", cmd_group[0])))
// 				c.Hub.Unicast <- mess
// 			}
// 		} else {
// 			mess := hub.NewMessage(c.CType, c, []byte("HELLO|?"))
// 			c.Hub.Unicast <- mess
// 		}
// 	case "[MNTR]":
// 		clog.Warn("server", "CallToActionTCP", "Wrong TCPManager: %s", cmd_group[0])
// 	default:
// 		clog.Warn("server", "CallToActionTCP", "Bad Command '%s', disconnecting client %s.", cmd_group[0], c.Name)
// 		c.Hub.Unregister <- c
// 		<-c.Consistent
// 	}
// }
