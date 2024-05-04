package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
)

const ipamDefaultAllocatorPath = "/var/lib/mydocker/network/ipam/subnet.json"

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             map[string]string // 网段和位图算法的数组 map, key 是网段， value 是分配的位图数组
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// Allocate 在网段中分配一个可用的 IP 地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = map[string]string{}
	if err := ipam.load(); err != nil {
		return nil, fmt.Errorf("load subnet allocation info error: %w", err)
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())
	one, size := subnet.Mask.Size()

	if _, exist := ipam.Subnets[subnet.String()]; !exist {
		ipam.Subnets[subnet.String()] = strings.Repeat("0", 1<<uint(size-one))
		if subnet.IP[3] == 0 {
			ipalloc := []byte(ipam.Subnets[subnet.String()])
			ipalloc[0] = '1'
			ipam.Subnets[subnet.String()] = string(ipalloc)
		}
	}

	for idx := range ipam.Subnets[subnet.String()] {
		if ipam.Subnets[subnet.String()][idx] == '0' {
			// 置为1， 表示已分配
			ipalloc := []byte(ipam.Subnets[subnet.String()])
			ipalloc[idx] = '1'
			ipam.Subnets[subnet.String()] = string(ipalloc)

			/*
				还需要通过网段的IP与上面的偏移相加计算出分配的IP地址，由于IP地址是uint的一个数组，
				需要通过数组中的每一项加所需要的值，比如网段是172.16.0.0/12，数组序号是65555,
				那么在[172,16,0,0] 上依次加[uint8(65555 >> 24)、uint8(65555 >> 16)、
				uint8(65555 >> 8)、uint8(65555 >> 0)]， 即[0, 1, 0, 19]， 那么获得的IP就
				是172.17.0.19.
			*/
			ip = subnet.IP
			for t := uint(4); t > 0; t-- {
				[]byte(ip)[4-t] += uint8(idx >> ((t - 1) * 8))
			}
			break
		}
	}
	if ip == nil {
		return ip, fmt.Errorf("the subnet ip addresses are used up")
	}
	if err = ipam.dump(); err != nil {
		logrus.Errorf("dump ipam subnets error: %v", err)
	}
	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = map[string]string{}
	if err := ipam.load(); err != nil {
		return fmt.Errorf("load subnet allocation info error: %w", err)
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())
	if _, ok := ipam.Subnets[subnet.String()]; !ok {
		return fmt.Errorf("invalid subnet: %v", subnet)
	}
	releaseIP := ipaddr.To4()
	var idx = 0
	for t := uint(4); t > 0; t-- {
		idx += int((releaseIP[t-1] - subnet.IP[t-1]) << ((4 - t) * 8))
	}

	ipalloc := []byte(ipam.Subnets[subnet.String()])
	ipalloc[idx] = '0'
	ipam.Subnets[subnet.String()] = string(ipalloc)

	err := ipam.dump()
	if err != nil {
		logrus.Errorf("dump ipam subnets error: %v", err)
	}
	return err
}

func (ipam *IPAM) load() error {
	// 检查存储文件状态，如果不存在，则说明之前没有分配，则不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	//读取文件，加载配置信息
	content, err := os.ReadFile(ipam.SubnetAllocatorPath)
	if err != nil {
		return fmt.Errorf("read subnets from config file error: %w", err)
	}
	if err = json.Unmarshal(content, &ipam.Subnets); err != nil {
		return fmt.Errorf("unmarshal subnets error: %w", err)
	}
	return nil
}

func (ipam *IPAM) dump() error {
	ipamConfigFolder, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFolder); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(ipamConfigFolder, constant.Perm0644); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
	if err != nil {
		return err
	}
	defer file.Close()
	subnetsBytes, err := json.MarshalIndent(ipam.Subnets, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal subents error: %w", err)
	}
	_, err = file.Write(subnetsBytes)
	return err

}
