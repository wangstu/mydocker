package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/wangstu/mydocker/container"
)

func ListContainers() {
	entries, err := os.ReadDir(container.InfoLoc)
	if err != nil {
		logrus.Errorf("read dir %s error: %v", container.InfoLoc, err)
		return
	}

	containerInfos := make([]*container.Info, 0, len(entries))
	for _, entry := range entries {
		tmpInfo, err := getContainerInfo(entry)
		if err != nil {
			logrus.Errorf("get container info error: %v", err)
			continue
		}
		containerInfos = append(containerInfos, tmpInfo)
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	if _, err = fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n"); err != nil {
		logrus.Errorf("fprint error: %v", err)
	}

	for _, item := range containerInfos {
		if _, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreateTime); err != nil {
			logrus.Errorf("fprint error: %v", err)
		}
	}
	w.Flush()
}

func getContainerInfo(entry os.DirEntry) (*container.Info, error) {
	folder := fmt.Sprintf(container.InfoLocFormat, entry.Name())
	infoFilePath := path.Join(folder, container.ConfigName)
	content, err := os.ReadFile(infoFilePath)
	if err != nil {
		logrus.Errorf("read file %s error: %v", infoFilePath, err)
		return nil, err
	}
	info := &container.Info{}
	if err = json.Unmarshal(content, info); err != nil {
		logrus.Errorf("json unmarshal error: %v", err)
		return nil, err
	}
	return info, nil
}
