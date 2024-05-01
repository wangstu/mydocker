package cmds

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	
	"github.com/wangstu/mydocker/container"
)

func GetContainerLog(containerId string) {
	logFilePath := path.Join(fmt.Sprintf(container.InfoLocFormat, containerId), container.GetLogFileName(containerId))
	file, err := os.Open(logFilePath)
	if err != nil && !os.IsNotExist(err) {
		logrus.Errorf("get container log error: %v", err)
		return
	}
	content, err := io.ReadAll(file)
	if err != nil {
		logrus.Errorf("read container log error: %v", err)
		return
	}
	if _, err = fmt.Fprint(os.Stdout, string(content)); err != nil {
		logrus.Errorf("print log error: %v", err)
	}

}
