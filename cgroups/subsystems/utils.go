package subsystems

import (
	"bufio"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/wangstu/mydocker/constant"
)

const mountPointIndex = 4

// getCgroupPath 找到cgroup在文件系统中的绝对路径
/*
	实际就是将根目录和cgroup名称拼接成一个路径。
	如果指定了自动创建，就先检测一下是否存在，如果对应的目录不存在，则说明cgroup不存在，这里就给创建一个
*/
func getCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := findCgroupMountPoint(subsystem)
	absPath := path.Join(cgroupRoot, cgroupPath)
	if !autoCreate {
		return absPath, nil
	}

	if _, err := os.Stat(absPath); err != nil && os.IsNotExist(err) {
		return absPath, os.Mkdir(absPath, constant.Perm0755)
	} else {
		return absPath, errors.Wrap(err, "create cgroup")
	}
}

// findCgroupMountPoint 通过 /proc/self/mountinfo 找出挂载了某个 subsystem 的 hierarchy cgroup 根节点所在的目录
func findCgroupMountPoint(subsystem string) string {
	// /proc/self/mountinfo 为当前进程的 mountinfo 信息
	// 可以通过 cat /proc/self/mountinfo 查看
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// txt 大概是这样的：104 85 0:20 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,memory
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		// 对最后一个元素按逗号进行分割，这里的最后一个元素就是 rw,memory
		// 其中的的 memory 就表示这是一个 memory subsystem
		subsystems := strings.Split(fields[len(fields)-1], ",")
		for _, opt := range subsystems {
			if opt == subsystem {
				// 如果等于指定的 subsystem，那么就返回这个挂载点跟目录，就是第四个元素，
				// 这里就是`/sys/fs/cgroup/memory`,即我们要找的根目录
				return fields[mountPointIndex]
			}
		}
	}

	if err = scanner.Err(); err != nil {
		logrus.Errorf("scan /proc/self/mountinfo error: %v", err)
	}
	return ""
}
