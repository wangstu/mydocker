package container

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
)

// NewParentProcess 构建 command 用于启动一个新进程
/*
	这里是父进程，也就是当前进程执行的内容。
	1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
	2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
	3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
	4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool, volume, containerId string) (*exec.Cmd, *os.File) {
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
	rootPath := "/home"
	newWorkSpace(rootPath, volume)
	// 指定 cmd 的工作目录为我们前面准备好的用于存放busybox rootfs的目录
	cmd.Dir = path.Join(rootPath, "merged")

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

func newWorkSpace(rootPath string, volume string) {
	createLower(rootPath)
	createDirs(rootPath)
	mountOverlayFS(rootPath)

	if volume != "" {
		mntPath := path.Join(rootPath, "merged")
		hostPath, containerPath, err := extractVolume(volume)
		if err != nil {
			logrus.Errorf("extract volume error: %v", err)
			return
		}
		mountVolume(mntPath, hostPath, containerPath)
	}
}

func createLower(rootPath string) {
	lower := path.Join(rootPath, "busybox")
	tarLower := path.Join(rootPath, "busybox.tar")
	logrus.Infof("lower path: %s, tar lower path: %s", lower, tarLower)

	exist, err := pathExists(lower)
	if err != nil {
		logrus.Errorf("check %s error: %v", lower, err)
	}
	if !exist {
		if err := os.Mkdir(lower, constant.Perm0777); err != nil {
			logrus.Errorf("mkdir %s error: %v", lower, err)
		}
		if _, err = exec.Command("tar", "-xvf", tarLower, "-C", lower).CombinedOutput(); err != nil {
			logrus.Errorf("uncompress %s error: %v", tarLower, err)
		}
	}
}

func createDirs(rootPath string) {
	dirs := []string{
		path.Join(rootPath, "merged"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"),
	}
	for _, dir := range dirs {
		if err := os.Mkdir(dir, constant.Perm0777); err != nil {
			logrus.Errorf("mkdir %s error: %v", dir, err)
		}
	}

}

func mountOverlayFS(rootPath string) {
	option := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		path.Join(rootPath, "busybox"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"))

	// mount -t overlay/ fuse.fuse-overlayfs overlay -o lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work /root/merged
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", option, path.Join(rootPath, "merged"))
	logrus.Infof("mount overlayfs: %s", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount overlay fs error: %v", err)
	}
}

func DeleteWorkSpace(rootPath, volume string) {
	// NOTE: 一定要要先 umount volume ，然后再删除目录，否则由于 bind mount 存在，删除临时目录会导致 volume 目录中的数据丢失。
	if volume != "" {
		_, containerPath, err := extractVolume(volume)
		if err != nil {
			logrus.Errorf("extract volume %s error: %v", volume, err)
		}
		umountVolume(path.Join(rootPath, "merged"), containerPath)
	}

	umountOverlayFS(rootPath)
	deleteDirs(rootPath)
}

func umountOverlayFS(rootPath string) {
	merged := path.Join(rootPath, "merged")
	cmd := exec.Command("umount", merged)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("umount %s error: %v", merged, err)
	}
}

func deleteDirs(rootPath string) {
	dirs := []string{
		path.Join(rootPath, "merged"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"),
	}

	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			logrus.Errorf("remove dir %s error %v", dir, err)
		} else {
			logrus.Infof("remove dir %s successfully", dir)
		}
	}
}

func pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
