package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const usage = `mydocker is a simple container runtime implementation.
               The purpose of this projevt is to learn how docker works and how to write a docker by ourseleves.
               Enjoy it, just for fun.`

func main() {
	app := cli.NewApp()
	app.Name = "mydocker"
	app.Usage = usage

	app.Commands = []cli.Command{
		initCmd,
		runCmd,
	}

	app.Before = func(ctx *cli.Context) error {
		// Log as JSON instead of the default ASCII formatter
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err) 
	}
}
