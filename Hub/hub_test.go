package Hub

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var tmpHub *Hub

func newClient(name string) *Client {
	tmpClient := &Client{
		Hub: tmpHub, Conn: "test", Disconnect: Diconnect, CallToAction: nil,
		CType: ClientUndefined, Send: make(chan []byte, 256),
		Identified: false, Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: "Test Socket", Mode: ReadWrite,
	}
	return tmpClient
}

func Diconnect(c *Client) {

}

func TestRegister(t *testing.T) {
	tmpClient := newClient("test")

	tmpHub.Register(tmpClient)

	assert.Equal(t, tmpClient, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name], "Registered Client should equal original Client")
}

func TestUnegister(t *testing.T) {
	tmpClient := newClient("test")

	tmpHub.Register(tmpClient)
	tmpHub.Unregister(tmpClient)

	assert.Nil(t, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name])
	assert.Nil(t, tmpClient.Hub)
}

func TestConcurrency(t *testing.T) {
	var tmpClient *Client

	for i := 0; i < 100; i++ {
		tmpClient = newClient(fmt.Sprintf("%d", i))

		go tmpHub.Register(tmpClient)
		tmpHub.clientProtect.Lock()
		assert.Equal(t, tmpClient, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name], "Registered Client should equal original Client")
		tmpHub.clientProtect.Unlock()
		go tmpHub.Unregister(tmpClient)

		tmpHub.clientProtect.Lock()
		assert.Nil(t, tmpHub.FullUsersList[tmpClient.CType][tmpClient.Name])
		tmpHub.clientProtect.Unlock()
	}

	// duration := time.Second * 5
	// time.Sleep(duration)
}

func TestMain(m *testing.M) {
	tmpHub = NewHub()
	m.Run()
}
