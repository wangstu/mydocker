package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"github.com/wangstu/mydocker/constant"
	"github.com/wangstu/mydocker/container"
)

var (
	defaultNetworkPath = "/var/lib/mydocker/network/network/"
	drivers            = map[string]Driver{}
)

func init() {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if !os.IsNotExist(err) {
			logrus.Errorf("check %s path error: %v", defaultNetworkPath, err)
			return
		}
		if err = os.MkdirAll(defaultNetworkPath, constant.Perm0644); err != nil {
			logrus.Errorf("create %s error: %v", defaultNetworkPath, err)
			return
		}
	}
}

func (net *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(dumpPath, constant.Perm0644); err != nil {
			return fmt.Errorf("create network dump path %s error: %w", defaultNetworkPath, err)
		}
	}

	netPath := path.Join(dumpPath, net.Name)
	netFile, err := os.OpenFile(netPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
	if err != nil {
		return fmt.Errorf("open file %s error: %w", netPath, err)
	}
	defer netFile.Close()

	netJson, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal network error: %w", err)
	}
	_, err = netFile.Write(netJson)
	return err
}

func (net *Network) remove(dumpPath string) error {
	netPath := path.Join(dumpPath, net.Name)
	if _, err := os.Stat(netPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.Remove(netPath)
}

func (net *Network) load(netPath string) error {
	content, err := os.ReadFile(netPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, net)
}

func loadNetworks() (map[string]*Network, error) {
	networks := map[string]*Network{}
	err := filepath.Walk(defaultNetworkPath, func(netPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		_, netName := path.Split(netPath)
		net := &Network{
			Name: netName,
		}
		if err = net.load(netPath); err != nil {
			logrus.Errorf("load network error: %v", err)
		}
		networks[netName] = net
		return nil
	})
	return networks, err
}

func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip

	net, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	return net.dump(defaultNetworkPath)
}

func ListNetwork() {
	networks, err := loadNetworks()
	if err != nil {
		logrus.Errorf("load networks from file error: %v", err)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, net := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			net.Name,
			net.IPRange.String(),
			net.Driver,
		)
	}
	w.Flush()
}

func DeleteNetwork(networkName string) error {
	networks, err := loadNetworks()
	if err != nil {
		return err
	}

	net, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("not found %s", networkName)
	}

	if err = ipAllocator.Release(net.IPRange, &net.IPRange.IP); err != nil {
		return fmt.Errorf("release ip error: %w", err)
	}
	if err = drivers[net.Driver].Delete(net); err != nil {
		return fmt.Errorf("remove network error: %w", err)
	}
	return net.remove(defaultNetworkPath)
}

func Connect(networkName string, info *container.Info) (net.IP, error) {
	networks, err := loadNetworks()
	if err != nil {
		return nil, err
	}

	network, ok := networks[networkName]
	if !ok {
		return nil, fmt.Errorf("not found %s", networkName)
	}

	// allocate container ip
	ip, err := ipAllocator.Allocate(network.IPRange)
	if err != nil {
		return ip, fmt.Errorf("allocate ip error: %w", err)
	}

	// create network endpoint
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", info.Id, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: info.PortMapping,
	}
	if err = drivers[network.Driver].Connect(network.Name, ep); err != nil {
		return ip, err
	}

	// configure container ip
	if err = configureEndpointIpAddressAndRoute(ep, info); err != nil {
		return ip, err
	}

	// configure port mapping
	return ip, addPortMapping(ep)
}

func Disconnect(info *container.Info) error {
	networks, err := loadNetworks()
	if err != nil {
		return err
	}

	networkName := info.NetworkName
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("not found %s", networkName)
	}

	drivers[network.Driver].Disconnect(fmt.Sprintf("%s-%s", info.Id, networkName))

	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", info.Id, networkName),
		IPAddress:   net.ParseIP(info.IP),
		Network:     network,
		PortMapping: info.PortMapping,
	}
	return deletePortMapping(ep)
}

func configureEndpointIpAddressAndRoute(ep *Endpoint, info *container.Info) error {
	// 根据名字找到对应Veth设备
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("get veth error: %w", err)
	}

	// 将容器的网络端点加入到容器的网络空间中
	// 并使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复为默认的网络空间
	defer enterContainerNetNS(&peerLink, info)()

	// set container veth ip
	ipNet := *ep.Network.IPRange
	ipNet.IP = ep.IPAddress
	if err = setInterfaceIP(ep.Device.PeerName, ipNet.String()); err != nil {
		return fmt.Errorf("set network %v error: %w", ep.Network, err)
	}
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return fmt.Errorf("set container veth up error: %w", err)
	}
	if err = setInterfaceUP("lo"); err != nil {
		return fmt.Errorf("set lo up error: %w", err)
	}

	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	// 构建要添加的路由数据，包括网络设备、网关IP及目的网段
	// 相当于route add -net 0.0.0.0/0 gw (Bridge网桥地址) dev （容器内的Veth端点设备)
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IPRange.IP,
		Dst:       cidr,
	}
	// 调用netlink的RouteAdd,添加路由到容器的网络空间
	// RouteAdd 函数相当于route add 命令
	return netlink.RouteAdd(defaultRoute)
}

func addPortMapping(ep *Endpoint) error {
	return configPortMapping(ep, false)
}

func deletePortMapping(ep *Endpoint) error {
	return configPortMapping(ep, true)
}

func configPortMapping(ep *Endpoint, isDelete bool) (err error) {
	action := "-A"
	if isDelete {
		action = "-D"
	}
	for _, pm := range ep.PortMapping {
		pms := strings.Split(pm, ":")
		if len(pms) != 2 {
			logrus.Errorf("port mapping format error: %v", pm)
			continue
		}
		// 由于iptables没有Go语言版本的实现，所以采用exec.Command的方式直接调用命令配置
		// 在iptables的PREROUTING中添加DNAT规则
		// 将宿主机的端口请求转发到容器的地址和端口上
		// iptables -t nat -A PREROUTING ! -i testbridge -p tcp -m tcp --dport 8080 -j DNAT --to-destination 10.0.0.4:8
		iptablesCmd := fmt.Sprintf("-t nat %s PREROUTING ! -i %s -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			action,
			ep.Network.Name,
			pms[0],
			ep.IPAddress.String(),
			pms[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		logrus.Infof("DNAT cmd: %v", cmd)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptable output: %v", string(output))
		}
	}
	return err
}

// enterContainerNetNS 将容器的网络端点加入到容器的网络空间中
// 并锁定当前程序所执行的线程，使当前线程进入到容器的网络空间
// 返回值是一个函数指针，执行这个返回函数才会退出容器的网络空间，回归到宿主机的网络空间
func enterContainerNetNS(link *netlink.Link, info *container.Info) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("get container net namespace error: %v", err)
	}
	nsFD := f.Fd()
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go语言的goroutine可能会被调度到别的线程上去
	// 就不能保证一直在所需要的网络空间中了
	// 所以先调用runtime.LockOSThread()锁定当前程序执行的线程
	runtime.LockOSThread()

	// 修改网络端点Veth的另外一端，将其移动到容器的Net Namespace 中
	if err = netlink.LinkSetNsFd(*link, int(nsFD)); err != nil {
		logrus.Errorf("set link ns error: %v", err)
	}

	originNs, err := netns.Get()
	if err != nil {
		logrus.Errorf("get current ns error: %v", err)
	}

	// 调用 netns.Set方法，将当前进程加入容器的Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("set netns error: %v", err)
	}

	// 在容器的网络空间中执行完容器配置之后调用此函数就可以将程序恢复到原生的Net Namespace
	return func() {
		// 恢复到上面获取到的之前的 Net Namespace
		netns.Set(originNs)
		originNs.Close()
		// 取消对当附程序的线程锁定
		runtime.UnlockOSThread()
		f.Close()
	}
}
