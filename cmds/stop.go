package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
	"github.com/wangstu/mydocker/container"
)

func StopContainer(containerId string) {
	// get container info
	containerInfo, err := getInfoByContainerId(containerId)
	if err != nil {
		logrus.Errorf("get container info error: %v", err)
		return
	}
	pidInt, err := strconv.Atoi(containerInfo.Pid)
	if err != nil {
		logrus.Errorf("convert pid error: %v", err)
		return
	}

	// send SIGTERM to container
	if err = syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		logrus.Errorf("stop container %s error: %v", containerId, err)
		return
	}

	// modify container info
	containerInfo.Pid = ""
	containerInfo.Status = container.STOP
	newContentBytes, err := json.MarshalIndent(containerInfo, "", "  ")
	if err != nil {
		logrus.Errorf("marshal json info error: %v", err)
		return
	}

	// save container info
	folder := fmt.Sprintf(container.InfoLocFormat, containerId)
	infoFilePath := path.Join(folder, container.ConfigName)
	if err = os.WriteFile(infoFilePath, newContentBytes, constant.Perm0644); err != nil {
		logrus.Errorf("write container info %s error: %v", infoFilePath, err)
	}
}

func getInfoByContainerId(containerId string) (*container.Info, error) {
	folder := fmt.Sprintf(container.InfoLocFormat, containerId)
	infoFilePath := path.Join(folder, container.ConfigName)
	contentBytes, err := os.ReadFile(infoFilePath)
	if err != nil {
		return nil, fmt.Errorf("read info error: %w", err)
	}
	containerInfo := &container.Info{}
	if err = json.Unmarshal(contentBytes, containerInfo); err != nil {
		return nil, fmt.Errorf("unmarshal info error: %w", err)
	}
	return containerInfo, nil
}
