package container

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
	"github.com/wangstu/mydocker/utils"
)

// NewParentProcess 构建 command 用于启动一个新进程
/*
	这里是父进程，也就是当前进程执行的内容。
	1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
	2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
	3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
	4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool, volume, containerId, imageName string) (*exec.Cmd, *os.File) {
	// 创建匿名管道用于传递参数，将readPipe作为子进程的ExtraFiles，子进程从readPipe中读取参数
	// 父进程中则通过writePipe将参数写入管道
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		logrus.Errorf("New pipe error: %v", err)
		return nil, nil
	}

	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	cmd.ExtraFiles = []*os.File{readPipe}

	// create overlay fs
	NewWorkSpace(containerId, imageName, volume)
	// 指定 cmd 的工作目录为我们前面准备好的用于存放busybox rootfs的目录
	cmd.Dir = utils.GetMergedPath(containerId)

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		folder := fmt.Sprintf(InfoLocFormat, containerId)
		if err := os.MkdirAll(folder, constant.Perm0644); err != nil && !os.IsExist(err) {
			logrus.Errorf("mkdir %s error: %v", folder, err)
			return nil, nil
		}
		logFilePath := path.Join(folder, GetLogFileName(containerId))
		logFile, err := os.Create(logFilePath)
		if err != nil {
			logrus.Errorf("create log file %s error: %v", logFilePath, err)
			return nil, nil
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	return cmd, writePipe
}
