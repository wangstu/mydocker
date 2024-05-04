package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/wangstu/mydocker/cgroups/subsystems"
	"github.com/wangstu/mydocker/cmds"
	"github.com/wangstu/mydocker/container"
	"github.com/wangstu/mydocker/network"
)

var runCmd = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgrups limit.
			mydocker run -d -name [containerName] [imageName] [command]`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
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
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment. eg: -e key=val",
		},
		cli.StringFlag{
			Name: "net",
			Usage: "container network. eg: -net test br",
		},
		cli.StringSliceFlag{
			Name: "p",
			Usage: "port mapping. eg: -p 8080:80, -p 30336:3306",
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
		detach := ctx.Bool("d")
		if tty && detach {
			return fmt.Errorf("it and d paramater can not both provided")
		}

		resourceConf := &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("mem"),
			CpuSet:      ctx.String("cpuset"),
			CpuCfsQuota: ctx.Int("cpu"),
		}
		volume := ctx.String("v")
		containerName := ctx.String("name")
		envSlice := ctx.StringSlice("e")
		networkName := ctx.String("net")
		portMapping := ctx.StringSlice("p")
		cmds.Run(tty, ctx.Args().Tail(), envSlice, resourceConf, volume, containerName, ctx.Args().First(), networkName, portMapping)
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
	Usage: "commit container to image. eg: mydocker commit iwue8390he myimage",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing container id and image name")
		}
		containerId := ctx.Args().Get(0)
		imageName := ctx.Args().Get(1)
		return cmds.Commit(containerId, imageName)
	},
}

var listCmd = cli.Command{
	Name:  "ps",
	Usage: "list containers",
	Action: func(ctx *cli.Context) error {
		cmds.ListContainers()
		return nil
	},
}

var logCmd = cli.Command{
	Name:  "logs",
	Usage: "print log of container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("please input container id")
		}
		containerId := ctx.Args().Get(0)
		cmds.GetContainerLog(containerId)
		return nil
	},
}

var execCmd = cli.Command{
	Name:  "exec",
	Usage: "exec a command into a container",
	Action: func(ctx *cli.Context) error {
		if os.Getenv(cmds.EnvExecPid) != "" {
			logrus.Infof("pid callback pid: %v", os.Getgid())
			return nil
		}
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerId := ctx.Args().Get(0)
		commands := ctx.Args().Tail()
		cmds.ExecContainer(containerId, commands)
		return nil
	},
}

var stopCmd = cli.Command{
	Name:  "stop",
	Usage: "stop container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container id")
		}
		cmds.StopContainer(ctx.Args().Get(0))
		return nil
	},
}

var rmCmd = cli.Command{
	Name:  "rm",
	Usage: "remove unused container",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "f",
			Usage: "force delete running container",
		},
	},
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container id")
		}
		containerId := ctx.Args().Get(0)
		force := ctx.Bool("f")
		cmds.RemoveContainer(containerId, force)
		return nil
	},
}

var networkCmd = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr block",
				},
			},
			Action: func(ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				driver := ctx.String("driver")
				subnet := ctx.String("subnet")
				name := ctx.Args().First()

				if err := network.CreateNetwork(driver, subnet, name); err != nil {
					return fmt.Errorf("create network error: %w", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(ctx *cli.Context) error {
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "rm",
			Usage: "delete container network",
			Action: func(ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				if err := network.DeleteNetwork(ctx.Args().First()); err != nil {
					return fmt.Errorf("remove network error: %w", err)
				}
				return nil
			},
		},
	},
}
