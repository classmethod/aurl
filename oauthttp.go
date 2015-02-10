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
	app.Commands = Commands

	app.Run(os.Args)
}
