package cmds

import (
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"
)

func Commit(imageName string) {
	mntPath := "/home/merged"
	tarImagePath := path.Join("/home", imageName+".tar")
	logrus.Infof("image tar path: %s", tarImagePath)
	if _, err := exec.Command("tar", "-czf", tarImagePath, "-C", mntPath, ".").CombinedOutput(); err != nil {
		logrus.Errorf("save conatainer image error: %v", err)
	}
}
