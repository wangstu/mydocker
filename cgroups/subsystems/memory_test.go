package subsystems

import (
	"os"
	"path"
	"testing"
)

func TestMemoryGroup(t *testing.T) {
	memSubSys := MemorySubSystem{}
	resConfig := &ResourceConfig{
		MemoryLimit: "1000m",
	}
	testCgroup := "testmemlimit"

	if err := memSubSys.Set(testCgroup, resConfig); err != nil {
		t.Fatalf("cgroup error: %v", err)
	}
	stat, _ := os.Stat(path.Join(findCgroupMountPoint("memory"), testCgroup))
	t.Logf("cgroup stats: %+v", stat)

	if err := memSubSys.Apply(testCgroup, os.Getegid(), resConfig); err != nil {
		t.Fatalf("cgroup apply %v", err)
	}
	t.Logf("add %d to %s", os.Getegid(), testCgroup)

	if err := memSubSys.Apply("", os.Getegid(), resConfig); err != nil {
		t.Fatalf("cgroup apply %v", err)
	}
	t.Logf("remove %d", os.Getegid())

	if err := memSubSys.Remove(testCgroup); err != nil {
		t.Fatalf("cgroup remove %v", err)
	}
}
