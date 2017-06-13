package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Djoulzy/Polycom/clog"
	"github.com/Djoulzy/Polycom/hub"
)

func HandShakeHTTP(c *hub.Client, message []byte) {
	h := c.Hub

	if string(message) == "Status:General" {
		clog.Info("server", "HandShakeHTTP", "New Status Client %s", c.Name)
		if len(h.Monitors) >= conf.MaxMonitorsConns {
			h.Unregister <- c
			<-c.Consistent
		} else {
			h.Newrole(&hub.ConnModifier{Client: c, NewName: c.Name, NewType: hub.ClientMonitor})
		}
	} else {
		uncrypted_message, _ := cryptor.Decrypt_b64(string(message))
		clog.Info("server", "HandShakeHTTP", "New User Client %s (%s)", c.Name, uncrypted_message)
		infos := strings.Split(string(uncrypted_message), "|")
		if len(infos) != 6 {
			clog.Warn("server", "HandShakeHTTP", "Bad Handshake format ... Disconnecting")
			h.Unregister <- c
			<-c.Consistent
			return
		}
		content_id, err := strconv.Atoi(strings.TrimSpace(infos[1]))
		if err != nil {
			clog.Warn("server", "HandShakeHTTP", "Unrecognized content_id ... Disconnecting")
			h.Unregister <- c
			<-c.Consistent
			return
		}

		actualName := c.Name
		newName := infos[0]

		// lock.Lock()
		if h.UserExists(actualName, hub.ClientUndefined) {
			if len(h.Users) >= conf.MaxUsersConns && !h.UserExists(newName, hub.ClientUser) {
				clog.Warn("server", "HandShakeHTTP", "Too many Users connections, rejecting %s (In:%d/Cl:%d).", actualName, len(h.Incomming), len(h.Users))
				if !scaleList.RedirectConnection(c) {
					clog.Error("server", "HandShakeHTTP", "NO FREE SLOTS !!!")
				}
				h.Unregister <- c
				<-c.Consistent
			} else {
				c.Hub.Newrole(&hub.ConnModifier{Client: c, NewName: newName, NewType: hub.ClientUser})
				c.Content_id = content_id
				c.Front_id = strings.TrimSpace(infos[2])
				c.App_id = strings.TrimSpace(infos[3])
				c.Country = strings.TrimSpace(infos[4])
				clog.Info("server", "HandShakeHTTP", "Identifying %s as %s", actualName, newName)

				scaleList.DispatchNewConnection(h, c.Name)
				// <-hub.Done
			}
		} else {
			clog.Warn("server", "HandShakeHTTP", "Can't identify client... Disconnecting %s.", c.Name)
			h.Unregister <- c
			<-c.Consistent
		}
		// lock.Unlock()
	}
}

func CallToActionHTTP(c *hub.Client, message []byte) {
	if c.CType != hub.ClientUndefined {
		if c.CType == hub.ClientUser {
			cmd_group := string(message[0:6])
			action_group := message[6:]
			switch cmd_group {
			case "[BCST]":
				mess := hub.NewMessage(hub.ClientUser, nil, action_group)
				c.Hub.Broadcast <- mess
			case "[UCST]":
			case "[CMMD]":
				Storage.NewRecord(string(action_group))
			case "[QUIT]":
				c.Hub.Unregister <- c
				<-c.Consistent
			}

		} else {
		}
	} else {
		HandShakeHTTP(c, message)
	}
}

func HandShakeTCP(c *hub.Client, cmd []string) {
	var ctype int

	name := cmd[1]
	if len(cmd) > 2 {
		ctype = hub.ClientServer
		if len(c.Hub.Servers) >= conf.MaxServersConns {
			clog.Warn("server", "HandShakeTCP", "Too many Server connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(c.Hub.Incomming), len(c.Hub.Servers))
			c.Hub.Unregister <- c
			<-c.Consistent
			return
		}
	} else {
		ctype = hub.ClientUser
		if len(c.Hub.Users) >= conf.MaxUsersConns {
			clog.Warn("server", "HandShakeTCP", "Too many Users connections, rejecting %s (In:%d/Cl:%d).", c.Name, len(c.Hub.Incomming), len(c.Hub.Users))
			c.Hub.Unregister <- c
			<-c.Consistent
			return
		}
	}

	if _, ok := c.Hub.Incomming[c.Name]; ok {
		clog.Info("server", "HandShakeTCP", "Identifying %s as %s", c.Name, name)
		c.Hub.Newrole(&hub.ConnModifier{Client: c, NewName: name, NewType: ctype})
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

func CallToActionTCP(c *hub.Client, message []byte) {
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
			if c.CType != hub.ClientUndefined {
				switch cmd_group[1] {
				case "quit":
					clog.Info("server", "CallToActionTCP", "Client %s deconnected normaly.", c.Name)
					c.Hub.Unregister <- c
					<-c.Consistent
				default:
					clog.Warn("server", "CallToActionTCP", "Unknown param %s for command %s", cmd_group[0], cmd_group[1])
					mess := hub.NewMessage(c.CType, c, []byte(fmt.Sprintf("%s:?", cmd_group[0])))
					c.Hub.Unicast <- mess
				}
			} else {
				mess := hub.NewMessage(c.CType, c, []byte("HELLO|?"))
				c.Hub.Unicast <- mess
			}
		case "MON":
			clog.Warn("server", "CallToActionTCP", "Wrong TCPManager: %s", cmd_group[0])
		default:
			clog.Warn("server", "CallToActionTCP", "Unknown Command: %s", cmd_group[0])
		}
	}
}
