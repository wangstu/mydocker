package cmds

import (
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/wangstu/mydocker/cgroups"
	"github.com/wangstu/mydocker/cgroups/subsystems"
	"github.com/wangstu/mydocker/container"
	"github.com/wangstu/mydocker/network"
)

func Run(tty bool, cmds, envSlice []string, res *subsystems.ResourceConfig,
	volume, containerName, imageName, networkName string, portMapping []string) {
	containerId := container.GenerateContainerID()

	parent, writePipe := container.NewParentProcess(tty, volume, containerId, imageName, envSlice)
	if parent == nil {
		logrus.Errorf("New Parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		logrus.Errorf("Run parent.Start error: %v", err)
		return
	}

	cgroupManager := cgroups.NewCgroupManager("mydocker", res)
	defer cgroupManager.Destory()
	_ = cgroupManager.Set()
	_ = cgroupManager.Apply(parent.Process.Pid)

	var containerIP string
	if networkName != "" {
		// config container network
		containerInfo := &container.Info{
			Id:          containerId,
			Pid:         strconv.Itoa(parent.Process.Pid),
			Name:        containerName,
			PortMapping: portMapping,
		}
		if ip, err := network.Connect(networkName, containerInfo); err != nil {
			logrus.Errorf("connect network error: %v", err)
			return
		} else {
			containerIP = ip.String()
			logrus.Infof("configured network, ip: %v", ip)
		}
	}

	containerInfo, err := container.RecordContainerInfo(parent.Process.Pid, cmds,
		containerName, containerId, volume, networkName, containerIP, portMapping)
	if err != nil {
		logrus.Errorf("record container info error: %v", err)
		return
	}

	sendInitCommands(writePipe, cmds)
	if tty {
		_ = parent.Wait()
		container.DeleteWorkSpace(containerId, volume)
		container.DeleteContainerInfo(containerId)
		if networkName != "" {
			network.Disconnect(containerInfo)
		}
	}
}

func sendInitCommands(writePipe *os.File, cmds []string) {
	cmdLine := strings.Join(cmds, " ")
	logrus.Infof("command line is: %v", cmdLine)
	writePipe.WriteString(cmdLine)
	writePipe.Close()
}
