package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/wangstu/mydocker/cgroups"
	"github.com/wangstu/mydocker/cgroups/subsystems"
	"github.com/wangstu/mydocker/container"
)

func Run(tty bool, cmds []string, res *subsystems.ResourceConfig, volume, containerName string) {
	containerId := container.GenerateContainerID()

	parent, writePipe := container.NewParentProcess(tty, volume)
	if parent == nil {
		logrus.Errorf("New Parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		logrus.Errorf("Run parent.Start error: %v", err)
		return
	}

	if err := container.RecordContainerInfo(parent.Process.Pid, cmds, containerName, containerId); err != nil {
		logrus.Errorf("record container info error: %v", err)
		return
	}

	cgroupManager := cgroups.NewCgroupManager("mydocker", res)
	defer cgroupManager.Destory()
	_ = cgroupManager.Set()
	_ = cgroupManager.Apply(parent.Process.Pid)

	sendInitCommands(writePipe, cmds)
	if tty {
		_ = parent.Wait()
		container.DeleteWorkSpace("/home", volume)
		container.DeleteContainerInfo(containerId)
	}
}

func sendInitCommands(writePipe *os.File, cmds []string) {
	cmdLine := strings.Join(cmds, " ")
	logrus.Infof("command line is: %v", cmdLine)
	writePipe.WriteString(cmdLine)
	writePipe.Close()
}
