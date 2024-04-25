package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/wangstu/mydocker/container"
)

func Run(tty bool, cmd string) {
	parent := container.NewParentProcess(tty, cmd)
	if err := parent.Start(); err != nil {
		logrus.Error(err)
	}
	parent.Wait()
	os.Exit(-1)
}
