package network

import (
	"net"

	"github.com/vishvananda/netlink"
)

type Network struct {
	Name    string
	IPRange *net.IPNet
	Driver  string
}

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

type Driver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(network *Network) error
	Connect(networkName string, endpoint *Endpoint) error
	Disconnect(endpointId string) error
}

type IPAMer interface {
	Allocate(subnet *net.IPNet) (ip *net.IP, err error)
	Release(subnet *net.IPNet, ipaddr *net.IP) error
}
