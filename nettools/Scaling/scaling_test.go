package scaling

import (
	"os"
	"testing"

	"github.com/Djoulzy/Polycom/Hub"

	clog "github.com/Djoulzy/Polycom/CLog"

	"github.com/Djoulzy/Polycom/nettools/TCPServer"
	"github.com/stretchr/testify/assert"
)

var tmpHub *Hub.Hub
var slist *ServersList

func TestAddServer(t *testing.T) {
	slist.AddNewPotentialServer("srv1", "127.0.0.1")
	slist.AddNewPotentialServer("srv1", "127.0.0.2")
	slist.AddNewPotentialServer("srv1", "127.0.0.3")
	slist.AddNewPotentialServer("srv1", "127.0.0.4")
	slist.AddNewPotentialServer("srv1", "127.0.0.5")

	assert.Equal(t, 4, len(slist.nodes), "Bad number of registred servers")
	// t.Errorf("%s", slist)
}

func TestMain(m *testing.M) {
	clog.LogLevel = 5
	clog.StartLogging = false

	tmpHub = Hub.NewHub()
	go tmpHub.Run()

	tcp_params := &TCPServer.Manager{
		ServerName:               "Test",
		Tcpaddr:                  "127.0.0.1",
		Hub:                      tmpHub,
		ConnectTimeOut:           2,
		WriteTimeOut:             1,
		ScalingCheckServerPeriod: 5,
		MaxServersConns:          5,
	}

	srvList := make(map[string]string)
	srvList["srv2"] = "127.0.0.3"
	srvList["srv2"] = "127.0.0.5"
	slist = Init(tcp_params, &srvList)
	os.Exit(m.Run())
}
