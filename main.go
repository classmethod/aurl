package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/classmethod/aurl/cli"
)

var (
	name       = "aurl"
	maintainer = "Seiichi Arai <arai.seiichi@classmethod.jp>"
	version    = "dev"
)

func main() {
	app := kingpin.New(name, fmt.Sprintf("Command line utility to call HTTP request with OAuth2.0. (version %s)", version))
	app.Version(version)
	app.Author(maintainer)

	a := cli.ConfigureGlobals(app)
	cli.ConfigureAddCommand(app, a)
	cli.ConfigureExecCommand(app, a)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
