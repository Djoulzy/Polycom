package Hub

import (
	"fmt"
	"os"
	"testing"

	clog "github.com/Djoulzy/Polycom/CLog"

	"github.com/stretchr/testify/assert"
)

var tmpHub *Hub

func newClient(name string, userType int) *Client {
	tmpClient := &Client{
		Hub: tmpHub, Conn: "NoC", Consistent: make(chan bool), Quit: make(chan bool, 8),
		CType: userType, Send: make(chan []byte, 256),
		CallToAction: nil, Addr: "127.0.0.1:8080",
		Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: "Test Socket",
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

	for i := 0; i < 100; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i), ClientUser)

		tmpHub.Register <- tmpClient
		<-tmpClient.Consistent
		assert.Equal(t, tmpClient, tmpHub.GetClientByName(tmpClient.Name, tmpClient.CType), "Registered Client should equal original Client")
		tmpHub.Unregister <- tmpClient
		<-tmpClient.Consistent
		assert.Nil(t, tmpHub.GetClientByName(tmpClient.Name, tmpClient.CType))
	}
}

func TestMessages(t *testing.T) {
	var tmpClient *Client

	for i := 0; i < 10; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i), ClientUser)

		tmpHub.Register <- tmpClient
		<-tmpClient.Consistent
	}

	mess := NewMessage(ClientUser, nil, []byte("BROADCAST"))
	tmpHub.Broadcast <- mess

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("%d", i)
		client := tmpHub.GetClientByName(name, ClientUser)

		message, ok := <-client.Send
		if ok {
			assert.Equal(t, "BROADCAST", string(message), "Message cannot be read from channel")
		} else {
			t.Fail()
		}
	}

	mess = NewMessage(ClientUser, tmpClient, []byte("UNICAST"))
	tmpHub.Unicast <- mess

	message, ok := <-tmpClient.Send
	if ok {
		assert.Equal(t, "UNICAST", string(message), "Message cannot be read from channel")
	} else {
		t.Fail()
	}
}

func TestNewRole(t *testing.T) {
	tmpClient := newClient("0", ClientUser)
	tmpHub.Register <- tmpClient
	<-tmpClient.Consistent

	newRole := &ConnModifier{
		Client:  tmpClient,
		NewName: "1",
		NewType: ClientUser,
	}

	tmpHub.Newrole(newRole)
	assert.Equal(t, tmpClient, tmpHub.GetClientByName("1", ClientUser), "Bad new Role")
	assert.Nil(t, tmpHub.GetClientByName("0", ClientUser))
}

func TestMain(m *testing.M) {
	clog.LogLevel = 5
	clog.StartLogging = true

	tmpHub = NewHub()
	go tmpHub.Run()
	os.Exit(m.Run())
}
