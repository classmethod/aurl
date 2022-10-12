package main

import (
	"os"
)

var (
	name    = "aurl"
	author  = "Seiichi Arai <arai.seiichi@classmethod.jp>"
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(cli.Run(os.Args))
}
