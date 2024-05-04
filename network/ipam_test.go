package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllocate(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("172.24.0.1/24")
	ip, err := ipAllocator.Allocate(ipNet)
	assert.Nil(t, err)
	t.Logf("alloc ip: %v", ip)
}

func TestRelease(t *testing.T) {
	ip, ipNet, _ := net.ParseCIDR("172.24.0.3/24")
	err := ipAllocator.Release(ipNet, &ip)
	assert.Nil(t, err)
}