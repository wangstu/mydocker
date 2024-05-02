package cmds

import (
	"fmt"
	"os"

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
		folder := fmt.Sprintf(container.InfoLocFormat, containerId)
		if err = os.RemoveAll(folder); err != nil {
			logrus.Errorf("remove container error: %v", err)
		}
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
