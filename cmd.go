package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangstu/mydocker/container"
)

var runCmd = cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "enable tty",
		},
	},

	/*
		1. chack command
		2. get conmmand
		3. execute
	*/
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container container command")
		}
		cmd := ctx.Args().Get(0)
		tty := ctx.Bool("it")
		Run(tty, cmd)
		return nil
	},
}

var initCmd = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside.",

	Action: func(ctx *cli.Context) error {
		log.Infof("init command")
		cmd := ctx.Args().Get(0)
		log.Infof("command: %s", cmd)
		return container.RunContainerInitProcess(cmd, nil)
	},
}
