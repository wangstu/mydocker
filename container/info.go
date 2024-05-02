package container

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/wangstu/mydocker/constant"
)

const (
	RUNNING       = "running"
	STOP          = "stopped"
	Exit          = "exited"
	InfoLoc       = "/var/lib/mydocker/containers/"
	InfoLocFormat = InfoLoc + "%s/"
	ConfigName    = "config.json"
	IDLength      = 10
)

type Info struct {
	Pid        string `json:"pid"`
	Id         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	CreateTime string `json:"createTime"`
	Status     string `json:"status"`
	Volume     string `json:"volume"`
}

func RecordContainerInfo(containerPID int, cmds []string, containerName, containerId, volume string) error {
	if containerName == "" {
		containerName = containerId
	}
	command := strings.Join(cmds, " ")
	containerInfo := &Info{
		Id:         containerId,
		Pid:        strconv.Itoa(containerPID),
		Command:    command,
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
		Status:     RUNNING,
		Name:       containerName,
		Volume:     volume,
	}
	jsonBytes, err := json.MarshalIndent(containerInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal container info error: %w", err)
	}
	jsonStr := string(jsonBytes)

	infoFolder := fmt.Sprintf(InfoLocFormat, containerId)
	if err := os.MkdirAll(infoFolder, constant.Perm0644); err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdir %s error: %w", infoFolder, err)
	}

	infoFilePath := path.Join(infoFolder, ConfigName)
	file, err := os.Create(infoFilePath)
	if err != nil {
		return fmt.Errorf("create file %s error: %w", infoFilePath, err)
	}
	if _, err = file.WriteString(jsonStr); err != nil {
		return fmt.Errorf("write container info to file %s error: %w", infoFilePath, err)
	}
	return nil
}

func DeleteContainerInfo(containerId string) error {
	infoFilePath := fmt.Sprintf(InfoLocFormat, containerId)
	if err := os.RemoveAll(infoFilePath); err != nil {
		return fmt.Errorf("remove %s error: %w", infoFilePath, err)
	}
	return nil
}

func GenerateContainerID() string {
	return randStringBytes(IDLength)
}

func randStringBytes(n int) string {
	letterBytes := "1234567890qwertyuiopasdfghjklzxcvbnm"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func GetLogFileName(containerId string) string {
	return containerId + "-json.log"
}
