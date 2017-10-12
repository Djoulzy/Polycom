package hub

import (
	"fmt"
	"os"
	"testing"

	"github.com/Djoulzy/Tools/clog"
	"github.com/stretchr/testify/assert"
)

var tmpHub *Hub

func newClient(name string, userType int) *Client {
	tmpClient := &Client{
		Quit:  make(chan bool, 8),
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
	assert.Equal(t, true, tmpHub.UserExists(tmpClient1.Name, tmpClient1.CType), "Client should be found")
	assert.Equal(t, tmpClient1, tmpHub.GetClientByName(tmpClient1.Name, tmpClient1.CType), "Registered Client should equal original Client")

	tmpHub.Register <- tmpClient2

	assert.Equal(t, true, tmpHub.UserExists(tmpClient1.Name, tmpClient1.CType), "Client should be found")
	assert.NotEqual(t, tmpClient1, tmpHub.GetClientByName(tmpClient1.Name, tmpClient1.CType), "Client should be replaced")
	assert.Equal(t, tmpClient2, tmpHub.GetClientByName(tmpClient1.Name, tmpClient1.CType), "Registered Client should equal to second client")
}

func TestUnregister(t *testing.T) {
	tmpClient := newClient("test", ClientUser)

	tmpHub.Register <- tmpClient
	tmpHub.Unregister <- tmpClient

	assert.Nil(t, tmpHub.GetClientByName(tmpClient.Name, tmpClient.CType))

	tmpClient3 := newClient("TestRegister", ClientServer)
	tmpHub.Register <- tmpClient3
	assert.Equal(t, tmpClient3, tmpHub.GetClientByName(tmpClient3.Name, tmpClient3.CType), "Registered Client should equal original Client")
	tmpHub.Register <- tmpClient3
}

func TestConcurrency(t *testing.T) {
	var tmpClient *Client

	for i := 0; i < 100; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i), ClientUser)

		tmpHub.Register <- tmpClient
		assert.Equal(t, tmpClient, tmpHub.GetClientByName(tmpClient.Name, tmpClient.CType), "Registered Client should equal original Client")
		tmpHub.Unregister <- tmpClient
		assert.Nil(t, tmpHub.GetClientByName(tmpClient.Name, tmpClient.CType))
	}
}

func TestMessages(t *testing.T) {
	var tmpClient *Client

	for i := 0; i < 10; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i), ClientUser)

		tmpHub.Register <- tmpClient
	}

	mess := NewMessage(tmpClient, ClientUser, nil, []byte("BROADCAST"))
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

	mess = NewMessage(nil, ClientUser, tmpClient, []byte("UNICAST"))
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
	clog.StartLogging = false

	tmpHub = NewHub()
	go tmpHub.Run()
	os.Exit(m.Run())
}
