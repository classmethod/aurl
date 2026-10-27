package main

import (
	"errors"
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
	app := kingpin.New(name, fmt.Sprintf("Command line utility to call HTTP request with OAuth2.0. (version %s)\nUsage: aurl <profile> <url>. Run `aurl help curl` for request flags (-X, -H, -d, ...).", version))
	app.Version(version)
	app.Author(maintainer)

	a := cli.ConfigureGlobals(app)
	cli.ConfigureAddCommand(app, a)
	cli.ConfigureExecCommand(app, a)
	cli.ConfigureMigrateCommand(app, a)

	command, err := app.Parse(os.Args[1:])
	if err != nil && (cli.IsV1Style(os.Args[1:]) || errors.Is(err, cli.ErrV1Style)) {
		app.Errorf("%s", err)
		fmt.Fprint(os.Stderr, cli.V1StyleNotice)
		os.Exit(1)
	}
	kingpin.MustParse(command, err)
}
