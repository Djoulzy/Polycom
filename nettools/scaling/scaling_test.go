package scaling

import (
	"os"
	"testing"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/nettools/tcpserver"
	"github.com/Djoulzy/Tools/clog"
	"github.com/stretchr/testify/assert"
)

var tmpHub *hub.Hub
var slist *ServersList

func newClient(name string, userType int) *hub.Client {
	tmpClient := &hub.Client{
		Quit:  make(chan bool),
		CType: userType, Send: make(chan []byte, 256),
		CallToAction: nil, Addr: "10.31.100.200:8081",
		Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: "Test Socket",
	}
	return tmpClient
}

func TestAddServer(t *testing.T) {
	slist.AddNewPotentialServer("srv1", "127.0.0.1:8081")
	slist.AddNewPotentialServer("srv1", "127.0.0.2:8081")
	slist.AddNewPotentialServer("srv1", "127.0.0.3:8081")
	slist.AddNewPotentialServer("srv1", "127.0.0.4:8081")
	slist.AddNewPotentialServer("srv1", "127.0.0.5:8081")

	assert.Equal(t, 5, len(slist.nodes), "Bad number of registred servers")
	// t.Errorf("%s", slist)
}

func TestAddNewConnectedServer(t *testing.T) {
	regSrv := newClient("test1", hub.ClientUndefined)
	tmpHub.Register <- regSrv
	tmpHub.Newrole(&hub.ConnModifier{Client: regSrv, NewName: "test1", NewType: hub.ClientServer})

	slist.AddNewConnectedServer(regSrv)
	assert.Equal(t, "test1", slist.nodes[regSrv.Addr].distantName, "Server should be registered")
}

// func TestCallToActionTCP(t *testing.T) {
// 	regSrv := slist.nodes["10.31.100.200:8081"].hubclient
// 	mess := "HELLO|VM|LISTN|10.31.100.200:8081"
// 	slist.CallToActionTCP(regSrv, []byte(mess))
// }

func TestUpdateMetrics(t *testing.T) {
	message := "{\"SID\":\"VM\",\"TCPADDR\":\"10.31.100.200:8081\",\"HTTPADDR\":\"10.31.100.200:8080\",\"HOST\":\"HTTP: 10.31.100.200:8080 - TCP: 10.31.100.200:8081\",\"CPU\":2,\"GORTNE\":8,\"STTME\":\"12/06/2017 11:45:18\",\"UPTME\":\"25.00085595s\",\"LSTUPDT\":\"12/06/2017 11:45:43\",\"LAVG\":5,\"MEM\":\"\u003cth\u003eMem\u003c/th\u003e\u003ctd class='memCell'\u003e3963 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e3653 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e6.8%\u003c/td\u003e\",\"SWAP\":\"\u003cth\u003eSwap\u003c/th\u003e\u003ctd class='memCell'\u003e1707 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e1707 Mo\u003c/td\u003e\u003ctd class='memCell'\u003e0.0%\u003c/td\u003e\",\"NBMESS\":1,\"NBI\":0,\"MXI\":500,\"NBU\":0,\"MXU\":200,\"NBM\":0,\"MXM\":3,\"NBS\":1,\"MXS\":5,\"BRTHLST\":{\"iMac\":{\"Tcpaddr\":\"10.31.200.168:8081\",\"Httpaddr\":\"localhost:8080\"}}}"

	go func() {
		<-monitoring.AddBrother
	}()

	slist.UpdateMetrics("10.31.100.200:8081", []byte(message))
	assert.Equal(t, 7, len(slist.nodes), "Bad number of registred servers")
	assert.Equal(t, "10.31.100.200:8080", slist.nodes["10.31.100.200:8081"].httpaddr, "Bad updated information")
}

func TestRedirectConnection(t *testing.T) {
	tmpClient := newClient("Toto", hub.ClientUser)
	tmpHub.Register <- tmpClient

	slist.RedirectConnection(tmpClient)
	ret := <-tmpClient.Send
	assert.Equal(t, "[RDCT]10.31.100.200:8080", string(ret), "Bad redirection data")
}

func TestMain(m *testing.M) {
	clog.LogLevel = 5
	clog.StartLogging = true

	tmpHub = hub.NewHub()
	go tmpHub.Run()

	tcp_params := &tcpserver.Manager{
		ServerName:               "Test",
		Tcpaddr:                  "127.0.0.1:8081",
		Hub:                      tmpHub,
		ConnectTimeOut:           2,
		WriteTimeOut:             1,
		ScalingCheckServerPeriod: 5,
		MaxServersConns:          5,
		CallToAction:             nil,
		Cryptor:                  nil,
	}

	srvList := make(map[string]string)
	srvList["srv2"] = "127.0.0.3"
	srvList["srv2"] = "127.0.0.5"
	slist = Init(tcp_params, &srvList)

	os.Exit(m.Run())
}
