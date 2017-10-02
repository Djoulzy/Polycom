package main

import (
	"fmt"
	"strings"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Tools/clog"
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
			if !ScaleList.RedirectConnection(c) {
				clog.Error("server", "welcomeNewUser", "NO FREE SLOTS !!!")
			}
			h.Unregister <- c
			<-c.Consistent
		} else {
			clog.Info("server", "welcomeNewUser", "Identifying %s as %s", c.Name, newName)
			h.Newrole(&hub.ConnModifier{Client: c, NewName: newName, NewType: hub.ClientUser})
			c.App_id = app_id
			ScaleList.DispatchNewConnection(h, c.Name)
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
		ScaleList.AddNewConnectedServer(c)
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
	clog.Test("", "", "%s", message)
	if len(message) < 6 {
		clog.Warn("server", "CallToAction", "Bad Command '%s', disconnecting client %s.", message, c.Name)
		h.Unregister <- c
		<-c.Consistent
		return
	}

	cmd_group := string(message[0:6])
	action_group := message[6:]

	if c.CType != hub.ClientUndefined {
		switch cmd_group {
		case "[BCST]":
			mess := hub.NewMessage(hub.ClientUser, nil, action_group)
			h.Broadcast <- mess
			if c.CType != hub.ClientServer {
				mess = hub.NewMessage(hub.ClientServer, nil, message)
				h.Broadcast <- mess
			}
		case "[UCST]":
		case "[STOR]":
			Storage.NewRecord(string(action_group))
		case "[QUIT]":
			h.Unregister <- c
			<-c.Consistent
		case "[MNIT]":
			clog.Debug("server", "CallToAction", "Metrics received from %s (%s)", c.Name, c.Addr)
			ScaleList.UpdateMetrics(c.Addr, action_group)
		case "[KILL]":
			id := string(action_group)
			if h.UserExists(id, hub.ClientUser) {
				userToKill := h.Users[id]
				clog.Info("server", "CallToAction", "Killing user %s", action_group)
				h.Unregister <- userToKill
				<-userToKill.Consistent
			}
		case "[GKEY]":
			crypted, _ := Cryptor.Encrypt_b64(string(action_group))
			mess := hub.NewMessage(c.CType, c, crypted)
			h.Unicast <- mess
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
