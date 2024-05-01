package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/wangstu/mydocker/cgroups/subsystems"
	"github.com/wangstu/mydocker/container"
)

var runCmd = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgrups limit.`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "mem",
			Usage: "memory limit. eg: --mem 100m",
		},
		cli.StringFlag{
			Name:  "cpu",
			Usage: "cpu quota. eg: --cpu 100",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit. eg: --cpuset 2,4",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "volume. eg: -v /etc/conf:/etc/conf",
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
		tty := ctx.Bool("it")
		resourceConf := &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("mem"),
			CpuSet:      ctx.String("cpuset"),
			CpuCfsQuota: ctx.Int("cpu"),
		}
		volume := ctx.String("v")
		Run(tty, ctx.Args(), resourceConf, volume)
		return nil
	},
}

var initCmd = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside.",

	Action: func(ctx *cli.Context) error {
		logrus.Infof("init command")
		cmd := ctx.Args().Get(0)
		logrus.Infof("command: %s", cmd)
		return container.RunContainerInitProcess()
	},
}

var commitCmd = cli.Command{
	Name:  "commit",
	Usage: "commit container to image",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing image name")
		}
		imageName := ctx.Args().Get(0)
		container.Commit(imageName)
		return nil
	},
}
