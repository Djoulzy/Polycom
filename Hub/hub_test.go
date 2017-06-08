package Hub

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var tmpHub *Hub

func newClient(name string) *Client {
	tmpClient := &Client{
		Hub: tmpHub, Conn: "NoC", Quit: make(chan bool),
		CType: ClientUndefined, Send: make(chan []byte, 256),
		CallToAction: nil, Addr: "127.0.0.1:8080",
		Identified: false, Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: "Test Socket", Mode: ReadWrite,
	}
	return tmpClient
}

func TestRegister(t *testing.T) {
	tmpClient := newClient("TestRegister")

	go tmpHub.Run()
	tmpHub.Register <- tmpClient
	tmpHub.Done <- true
	// duration := time.Second / 10
	// time.Sleep(duration)
	assert.Equal(t, tmpClient, tmpHub.GetClientByName(tmpClient.CType, tmpClient.Name), "Registered Client should equal original Client")
}

func TestUnegister(t *testing.T) {
	tmpClient := newClient("test")

	go tmpHub.Run()
	tmpHub.Register <- tmpClient
	tmpHub.Unregister <- tmpClient
	tmpHub.Done <- true

	assert.Nil(t, tmpHub.GetClientByName(tmpClient.CType, tmpClient.Name))
	assert.Nil(t, tmpClient.Hub)
}

func TestConcurrency(t *testing.T) {
	var tmpClient *Client

	go tmpHub.Run()
	for i := 0; i < 100; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i))

		tmpHub.Register <- tmpClient
		assert.Equal(t, tmpClient, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name], "Registered Client should equal original Client")
		tmpHub.Unregister <- tmpClient
		assert.Nil(t, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name])
	}
	tmpHub.Done <- true
}

func TestMain(m *testing.M) {
	tmpHub = NewHub()
	os.Exit(m.Run())
}
