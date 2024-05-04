package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const defaultPeerVethName = "eth0"

type BridgeNetworkDriver struct{}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetworkDriver) Create(subnet, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	network := &Network{
		Name:    name,
		IPRange: ipRange,
		Driver:  d.Name(),
	}

	// configure linux bridge
	if err := d.initBridge(network); err != nil {
		return nil, fmt.Errorf("create linux bridge network error: %w", err)
	}
	return network, nil
}

func (d *BridgeNetworkDriver) Delete(network *Network) error {
	if err := deleteIPRoute(network.Name, network.IPRange.String()); err != nil {
		return fmt.Errorf("clean route rule error: %v", err)
	}

	if err := deleteIPTables(network.Name, network.IPRange); err != nil {
		return fmt.Errorf("clean snat iptables rule error: %v", err)
	}
	// 删除网桥
	if err := d.deleteBridge(network); err != nil {
		return fmt.Errorf("delete bridge %v error: %v", network.Name, err)
	}
	return nil
}

// deleteBridge deletes the bridge
func (d *BridgeNetworkDriver) deleteBridge(n *Network) error {
	bridgeName := n.Name

	// get the link
	l, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("getting link with name %s error: %w", bridgeName, err)
	}

	// delete the link
	if err = netlink.LinkDel(l); err != nil {
		return fmt.Errorf("delete network interface %s error: %w", bridgeName, err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Connect(networkName string, endpoint *Endpoint) error {
	bridgeName := networkName
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	linkAttr := netlink.NewLinkAttrs()
	linkAttr.Name = endpoint.ID[:5]
	// 通过设置 Veth 接口 master 属性，设置这个Veth的一端挂载到网络对应的 Linux Bridge
	linkAttr.MasterIndex = bridge.Attrs().Index
	// 创建 Veth 对象，通过 PeerNarne 配置 Veth 另外一端的接口名
	endpoint.Device = netlink.Veth{
		LinkAttrs: linkAttr,
		PeerName:  defaultPeerVethName,
	}

	// 调用netlink的LinkAdd方法创建出这个Veth接口
	// 因为上面指定了link的MasterIndex是网络对应的Linux Bridge
	// 所以Veth的一端就已经挂载到了网络对应的LinuxBridge.上
	if err := netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("add endpoint device error: %w", err)
	}

	// 调用netlink的LinkSetUp方法，设置Veth启动
	// 相当于ip link set xxx up命令
	if err := netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("set endpoint device up error: %w", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(endpointId string) error {
	vethName := endpointId[:5]
	veth, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("get veth %s error: %w", vethName, err)
	}
	if err := netlink.LinkSetNoMaster(veth); err != nil {
		return err
	}

	// del veth-pair
	err = netlink.LinkDel(veth)
	if err != nil {
		return fmt.Errorf("delete veth %s error: %w", vethName, err)
	}
	peerVeth, err := netlink.LinkByName(defaultPeerVethName)
	if err != nil {
		return fmt.Errorf("get peer veth %s error: %w", defaultPeerVethName, err)
	}
	err = netlink.LinkDel(peerVeth)
	if err != nil {
		return fmt.Errorf("delete peer veth %s error: %w", defaultPeerVethName, err)
	}
	return nil
}

// initBridge 初始化Linux Bridge
/*
Linux Bridge 初始化流程如下：
1）创建 Bridge 虚拟设备
2）设置 Bridge 设备地址和路由
3）启动 Bridge 设备
4）设置 iptables SNAT 规则
*/
func (d *BridgeNetworkDriver) initBridge(network *Network) error {
	bridgeName := network.Name

	// 1. create bridge virtual device
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("create bridge error: %w", err)
	}

	// 2. configure bridge address and route
	if err := setInterfaceIP(bridgeName, network.IPRange.String()); err != nil {
		return fmt.Errorf("set bridge %s ip error: %w", bridgeName, err)
	}

	// 3. set bridge up
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("set bridge %s up error: %w", bridgeName, err)
	}

	// 4. configure iptable snat
	if err := setupIPTables(bridgeName, network.IPRange); err != nil {
		return fmt.Errorf("set snat of bridge %s error: %w", bridgeName, err)
	}

	return nil
}

// createBridgeInterface 创建Bridge设备
// ip link add xxxx
func createBridgeInterface(bridgeName string) error {
	// check bridge device by name
	if _, err := net.InterfaceByName(bridgeName); err != nil && !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// create bridge object
	linkAttr := netlink.NewLinkAttrs()
	linkAttr.Name = bridgeName
	bridge := &netlink.Bridge{LinkAttrs: linkAttr}
	// netlink.LinkAdd creates virtual bridge(ip link add xxx)
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("create bridge error: %w", err)
	}
	return nil
}

// Set the IP addr of a netlink interface
// ip addr add xxx命令
func setInterfaceIP(name string, cidrBlock string) error {
	retryTimes := 2
	var err error
	var iface netlink.Link
	for i := 0; i < retryTimes; i++ {
		if iface, err = netlink.LinkByName(name); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("get bridge error: %w", err)
	}

	ipNet, err := netlink.ParseIPNet(cidrBlock)
	if err != nil {
		return err
	}
	// ip addr add xxx
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

// setInterfaceUP 启动Bridge设备
// 等价于 ip link set xxx up 命令
func setInterfaceUP(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("get interface %s error: %w", name, err)
	}

	// ip link set xxx up
	if err = netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("enable %s error: %w", name, err)
	}
	return nil
}

// setupIPTables 设置 iptables 对应 bridge MASQUERADE 规则
// iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -o eth0 -j MASQUERADE
// iptables -t nat -A POSTROUTING -s {subnet} -o {deviceName} -j MASQUERADE
func setupIPTables(bridgeName string, ipNet *net.IPNet) error {
	return configIPTables(bridgeName, ipNet, false)
}

func deleteIPTables(bridgeName string, ipNet *net.IPNet) error {
	return configIPTables(bridgeName, ipNet, true)
}

func configIPTables(name string, ipNet *net.IPNet, isDelete bool) error {
	action := "-A"
	if isDelete {
		action = "-D"
	}
	iptabelCmd := fmt.Sprintf("-t nat %s POSTROUTING -s %s ! -o %s -j MASQUERADE", action, ipNet.String(), name)
	cmd := exec.Command("iptables", strings.Split(iptabelCmd, " ")...)
	logrus.Infof("set iptable command: %v", cmd.String())
	if output, err := cmd.Output(); err != nil {
		return fmt.Errorf("iptables error: %w, output: %v", err, output)
	}
	return nil
}

// ip addr del xxx
func deleteIPRoute(name string, cidrBlock string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		logrus.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("get network interface error: %w", err)
	}

	list, err := netlink.RouteList(iface, netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	for _, route := range list {
		if route.Dst.String() == cidrBlock { // 根据子网进行匹配
			err = netlink.RouteDel(&route)
			if err != nil {
				logrus.Errorf("del route %v error:%v", route, err)
				continue
			}
		}
	}
	return nil
}
