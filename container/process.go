package container

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
)

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
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
	newWorkSpace(rootPath)
	// 指定 cmd 的工作目录为我们前面准备好的用于存放busybox rootfs的目录
	cmd.Dir = filepath.Join(rootPath, "merged")

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd, writePipe
}

func newWorkSpace(rootPath string) {
	createLower(rootPath)
	createDirs(rootPath)
	mountOverlayFS(rootPath)
}

func createLower(rootPath string) {
	lower := filepath.Join(rootPath, "busybox")
	tarLower := filepath.Join(rootPath, "busybox.tar")
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
		filepath.Join(rootPath, "merged"),
		filepath.Join(rootPath, "upper"),
		filepath.Join(rootPath, "work"),
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
		filepath.Join(rootPath, "upper"),
		filepath.Join(rootPath, "work"))

	// mount -t overlay/ fuse.fuse-overlayfs overlay -o lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work /root/merged
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", option, filepath.Join(rootPath, "merged"))
	logrus.Infof("mount overlayfs: %s", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount overlay fs error: %v", err)
	}
}

func DeleteWorkSpace(rootPath string) {
	umountOverlayFS(rootPath)
	deleteDirs(rootPath)
}

func umountOverlayFS(rootPath string) {
	merged := filepath.Join(rootPath, "merged")
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
