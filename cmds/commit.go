package cmds

import (
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/utils"
)

func Commit(containerId, imageName string) error {
	mntPath := utils.GetMergedPath(containerId)
	tarImagePath := utils.GetImagePath(imageName)
	exist, err := utils.IsPathExist(tarImagePath)
	if err != nil {
		return fmt.Errorf("check image path %s error: %w", tarImagePath, err)
	}
	if exist {
		logrus.Warnf("%s is existed, overwrite it", tarImagePath)
	}

	logrus.Infof("image tar path: %s", tarImagePath)
	if _, err := exec.Command("tar", "-czf", tarImagePath, "-C", mntPath, ".").CombinedOutput(); err != nil {
		logrus.Errorf("save conatainer image error: %v", err)
	}
	return nil
}
