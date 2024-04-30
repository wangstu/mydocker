package subsystems

import "testing"

func TestFindCgroupMountpoint(t *testing.T) {
	t.Logf("cpu subsystem mount point: %v\n", findCgroupMountPoint("cpu"))
	t.Logf("cpuset subsystem mount point: %v\n", findCgroupMountPoint("cpuset"))
	t.Logf("memory subsystem mount point: %v\n", findCgroupMountPoint("memory"))
}
