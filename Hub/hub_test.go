package Hub

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var tmpHub *Hub

func newClient(name string, userType int) *Client {
	tmpClient := &Client{
		Hub: tmpHub, Conn: "NoC", Consistent: make(chan bool), Quit: make(chan bool, 8),
		CType: userType, Send: make(chan []byte, 256),
		CallToAction: nil, Addr: "127.0.0.1:8080",
		Identified: false, Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: "Test Socket", Mode: ReadWrite,
	}
	return tmpClient
}

func TestRegister(t *testing.T) {
	tmpClient1 := newClient("TestRegister", ClientUser)
	tmpClient2 := newClient("TestRegister", ClientUser)

	tmpHub.Register <- tmpClient1
	<-tmpClient1.Consistent
	assert.Equal(t, true, tmpHub.UserExists(tmpClient1.Name, tmpClient1.CType), "Client should be found")
	assert.Equal(t, tmpClient1, tmpHub.GetClientByName(tmpClient1.Name, tmpClient1.CType), "Registered Client should equal original Client")

	tmpHub.Register <- tmpClient2
	<-tmpClient2.Consistent

	assert.Equal(t, true, tmpHub.UserExists(tmpClient1.Name, tmpClient1.CType), "Client should be found")
	assert.NotEqual(t, tmpClient1, tmpHub.GetClientByName(tmpClient1.Name, tmpClient1.CType), "Client should be replaced")
	assert.Equal(t, tmpClient2, tmpHub.GetClientByName(tmpClient1.Name, tmpClient1.CType), "Registered Client should equal to second client")
}

func TestUnregister(t *testing.T) {
	tmpClient := newClient("test", ClientUser)

	tmpHub.Register <- tmpClient
	<-tmpClient.Consistent
	tmpHub.Unregister <- tmpClient
	<-tmpClient.Consistent

	assert.Nil(t, tmpHub.GetClientByName(tmpClient.Name, tmpClient.CType))
	assert.Nil(t, tmpClient.Hub)

	tmpClient3 := newClient("TestRegister", ClientServer)
	tmpHub.Register <- tmpClient3
	<-tmpClient3.Consistent
	assert.Equal(t, tmpClient3, tmpHub.GetClientByName(tmpClient3.Name, tmpClient3.CType), "Registered Client should equal original Client")
	tmpHub.Register <- tmpClient3
	<-tmpClient3.Consistent
	assert.Nil(t, tmpClient3.Hub)
}

func TestConcurrency(t *testing.T) {
	var tmpClient *Client

	go tmpHub.Run()
	for i := 0; i < 100; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i), ClientUser)

		tmpHub.Register <- tmpClient
		<-tmpClient.Consistent
		assert.Equal(t, tmpClient, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name], "Registered Client should equal original Client")
		tmpHub.Unregister <- tmpClient
		<-tmpClient.Consistent
		assert.Nil(t, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name])
	}
}

func TestMain(m *testing.M) {
	tmpHub = NewHub()
	go tmpHub.Run()
	os.Exit(m.Run())
}
