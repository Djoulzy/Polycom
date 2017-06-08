package scaling

import (
	"os"
	"testing"

	"github.com/Djoulzy/Polycom/Hub"
	"github.com/Djoulzy/Polycom/nettools/TCPServer"
	"github.com/stretchr/testify/assert"
)

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
	hub := Hub.NewHub()
	tcp_params := &TCPServer.Manager{
		ServerName:               "Test",
		Tcpaddr:                  "127.0.0.1",
		Hub:                      hub,
		ConnectTimeOut:           2,
		WriteTimeOut:             1,
		ScalingCheckServerPeriod: 5,
		MaxServersConns:          5,
	}

	slist = Init(tcp_params, nil)
	os.Exit(m.Run())
}
