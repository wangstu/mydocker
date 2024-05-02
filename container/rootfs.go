package container

import (
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/constant"
	"github.com/wangstu/mydocker/utils"
)

// NewWorkSpace Create an Overlay2 filesystem as container root workspace
/*
1）创建lower层
2）创建upper、worker层
3）创建merged目录并挂载overlayFS
4）如果有指定volume则挂载volume
*/
func NewWorkSpace(containerId, imageName, volume string) {
	createLower(containerId, imageName)
	createDirs(containerId)
	mountOverlayFS(containerId)

	if volume != "" {
		mntPath := utils.GetMergedPath(containerId)
		hostPath, containerPath, err := extractVolume(volume)
		if err != nil {
			logrus.Errorf("extract volume error: %v", err)
			return
		}
		mountVolume(mntPath, hostPath, containerPath)
	}
}

func createLower(containerId, imageName string) {
	lower := utils.GetLowerPath(containerId)
	tarLower := utils.GetImagePath(imageName)
	logrus.Infof("lower path: %s, tar lower path: %s", lower, tarLower)

	exist, err := utils.IsPathExist(lower)
	if err != nil {
		logrus.Errorf("check %s error: %v", lower, err)
	}
	if !exist {
		if err := os.MkdirAll(lower, constant.Perm0777); err != nil {
			logrus.Errorf("mkdir %s error: %v", lower, err)
		}
		if _, err = exec.Command("tar", "-xvf", tarLower, "-C", lower).CombinedOutput(); err != nil {
			logrus.Errorf("uncompress %s error: %v", tarLower, err)
		}
	}
}

func createDirs(containerId string) {
	dirs := []string{
		utils.GetMergedPath(containerId),
		utils.GetUpperPath(containerId),
		utils.GetWorkPath(containerId),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, constant.Perm0777); err != nil {
			logrus.Errorf("mkdir %s error: %v", dir, err)
		}
	}
}

func mountOverlayFS(containerId string) {
	option := utils.GetMountOption(containerId)
	mergedPath := utils.GetMergedPath(containerId)

	// mount -t overlay/ fuse.fuse-overlayfs overlay -o lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work /root/merged
	driverType := "overlay"
	if os.Getenv("DRIVER_TYPE") != "" {
		driverType = os.Getenv("DRIVER_TYPE")
	}
	cmd := exec.Command("mount", "-t", driverType, "overlay", "-o", option, mergedPath)
	logrus.Infof("mount overlayfs: %s", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount overlay fs error: %v", err)
	}
}

// DeleteWorkSpace Delete the UFS filesystem while container exit
/*
和创建相反
1）有volume则卸载volume
2）卸载并移除merged目录
3）卸载并移除upper、worker层
*/
func DeleteWorkSpace(containerId, volume string) {
	// NOTE: 一定要先 umount volume ，然后再删除目录，否则由于 bind mount 存在，删除临时目录会导致 volume 目录中的数据丢失。
	if volume != "" {
		_, containerPath, err := extractVolume(volume)
		if err != nil {
			logrus.Errorf("extract volume %s error: %v", volume, err)
		}
		mntPath := utils.GetMergedPath(containerId)
		umountVolume(mntPath, containerPath)
	}

	umountOverlayFS(containerId)
	deleteDirs(containerId)
}

func umountOverlayFS(containerId string) {
	mergedPath := utils.GetMergedPath(containerId)
	cmd := exec.Command("umount", mergedPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logrus.Infof("umount merged path: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		logrus.Errorf("umount %s error: %v", mergedPath, err)
	}
}

func deleteDirs(containerId string) {
	dirs := []string{
		utils.GetMergedPath(containerId),
		utils.GetUpperPath(containerId),
		utils.GetWorkPath(containerId),
		utils.GetLowerPath(containerId),
		utils.GetRootPath(containerId),
	}

	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			logrus.Errorf("remove dir %s error %v", dir, err)
		} else {
			logrus.Infof("remove dir %s successfully", dir)
		}
	}
}
