package cgroups

import (
	"github.com/sirupsen/logrus"
	
	"github.com/wangstu/mydocker/cgroups/subsystems"
)

type CgroupManager struct {
	Path     string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string, res *subsystems.ResourceConfig) *CgroupManager {
	return &CgroupManager{
		Path:     path,
		Resource: res,
	}
}

func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Apply(c.Path, pid, c.Resource); err != nil {
			logrus.Errorf("apply subsystem %s error: %s", subSysIns.Name(), err)
		}
	}
	return nil
}

func (c *CgroupManager) Set() error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Set(c.Path, c.Resource); err != nil {
			logrus.Errorf("apply subsystem %s error: %v", subSysIns.Name(), err)
		}
	}
	return nil
}

func (c *CgroupManager) Destory() error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			logrus.Warnf("remove cgroup error: %v", err)
		}
	}
	return nil
}
