package container

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
)

func mountVolume(mntPath, hostPath, containerPath string) {
	if err := os.Mkdir(hostPath, constant.Perm0777); err != nil && !os.IsExist(err) {
		logrus.Errorf("mkdir host dir %s error: %v", hostPath, err)
		return
	}

	// 在 merged 中创建对应的目录
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := os.Mkdir(containerPathInHost, constant.Perm0777); err != nil && !os.IsExist(err) {
		logrus.Errorf("mkdir container dir %s error: %v", containerPathInHost, err)
		return
	}

	// bind mount: mount -o bind /hostPath /containerPathInHost
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount volume error: %v", err)
	}
}

func umountVolume(mntPath, containerPath string) {
	containerPathInHost := path.Join(mntPath, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("umount volume error: %v", err)
	}
}

func extractVolume(volume string) (hostPath, containerPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		err = fmt.Errorf("invalid volume: %s, must be splited by `:`", volume)
		return
	}

	hostPath, containerPath = parts[0], parts[1]
	if !isValidPath(hostPath) || !isValidPath(containerPath) {
		err = fmt.Errorf("invalid volume: %s", volume)
	}
	return
}

func isValidPath(p string) bool {
	if p == "" {
		return false
	}
	return path.IsAbs(p)
}
