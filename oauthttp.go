package main

import (
	"os"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "oauthttp"
	app.Version = Version
	app.Usage = ""
	app.Author = "Daisuke Miyamoto"
	app.Email = "miyamoto.daisuke@classmethod.jp"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "profile",
			Value:  "default",
			Usage:  "profile name",
			EnvVar: "OAUTHTTP_PROFILE",
		},
	}
	app.Commands = Commands

	app.Run(os.Args)
}
