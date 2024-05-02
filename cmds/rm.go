package cmds

import (
	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/container"
)

func RemoveContainer(containerId string, force bool) {
	containerInfo, err := getInfoByContainerId(containerId)
	if err != nil {
		logrus.Errorf("get container info error: %v", err)
		return
	}

	switch containerInfo.Status {
	case container.STOP:
		if err = container.DeleteContainerInfo(containerId); err != nil {
			logrus.Errorf("remove container %s config error: %v", containerId, err)
			return
		}
		container.DeleteWorkSpace(containerId, containerInfo.Volume)
	case container.RUNNING:
		if !force {
			logrus.Errorf("can't remove running container %s, please stop container before attempting removal or force to remove", containerId)
			return
		}
		logrus.Infof("force to delete container: %s", containerId)
		StopContainer(containerId)
		RemoveContainer(containerId, force)
	default:
		logrus.Errorf("container %s is in invalid status: %s", containerId, containerInfo.Status)
	}
}
