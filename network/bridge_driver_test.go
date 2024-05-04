package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

var name = "testBridge"

func TestCreateBridge(t *testing.T) {
	bridge := BridgeNetworkDriver{}
	net, err := bridge.Create("172.24.0.1/24", name)
	assert.Nil(t, err)
	t.Logf("create network: %v", net)
}

func TestDeleteBridge(t *testing.T) {
	bridge := BridgeNetworkDriver{}
	_, ipRange, _ := net.ParseCIDR("192.168.0.1/24")
	network := &Network{
		Name:    name,
		IPRange: ipRange,
	}
	assert.Nil(t, bridge.Delete(network))
}

func TestConnect(t *testing.T) {
	ep := &Endpoint{
		ID: "test-container",
	}
	net := &Network{Name: name}
	bridge := BridgeNetworkDriver{}
	assert.Nil(t, bridge.Connect(net.Name, ep))
}

func TestDisconnect(t *testing.T) {
	ep := &Endpoint{
		ID: "test-container",
	}
	bridge := BridgeNetworkDriver{}
	assert.Nil(t, bridge.Disconnect(ep.ID))
}
