package container

import (
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
)

func RunContainerInitProcess(command string, args []string) error {
	logrus.Infof("command: %s", command)
	
	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示声明你要这个新的mount namespace独立。
	// 即 mount proc 之前先把所有挂载点的传播类型改为 private，避免本 namespace 中的挂载事件外泄。
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	
	// 如果不先做 private mount，会导致挂载事件外泄，后续再执行 mydocker 命令时 /proc 文件系统异常
	// 可以执行 mount -t proc proc /proc 命令重新挂载来解决
	defaultMountFlag := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlag), "")
	argv := []string{command}
	if err := syscall.Exec(command, argv, os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}
