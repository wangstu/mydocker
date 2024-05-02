package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/wangstu/mydocker/container"
	_ "github.com/wangstu/mydocker/nsenter"
)

const (
	EnvExecPid = "mydocker_pid"
	EnvExecCmd = "mydocker_cmd"
)

// nsenter里的C代码里已经出现mydocker_pid和mydocker_cmd这两个Key,主要是为了控制是否执行C代码里面的setns.
func ExecContainer(containerId string, cmds []string) {
	pid, err := getPidByContainerId(containerId)
	if err != nil {
		logrus.Errorf("get pid from container id error: %v", err)
		return
	}

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmdStr := strings.Join(cmds, " ")
	logrus.Infof("container pid: %s, command: %s", pid, cmdStr)
	os.Setenv(EnvExecPid, pid)
	os.Setenv(EnvExecCmd, cmdStr)

	if err = cmd.Run(); err != nil {
		logrus.Errorf("exec container %s error: %v", containerId, err)
	}
}

func getPidByContainerId(containerId string) (string, error) {
	folder := fmt.Sprintf(container.InfoLocFormat, containerId)
	infoFilePath := path.Join(folder, container.ConfigName)
	contentBytes, err := os.ReadFile(infoFilePath)
	if err != nil {
		return "", err
	}
	containerInfo := &container.Info{}
	if err := json.Unmarshal(contentBytes, containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}
