package main

import (
	"github.com/codegangsta/cli"
	"log"
	"os"
)

var CurrentOptions Options

// Options provides a nice container to hold the values of command line options
type Options struct {
	ProfileName string
	Verbose     bool
	ProfileDict map[string]map[string]string
}

// extract an Options instance from the command line arguments
func Opts(c *cli.Context) (Options, error) {
	if dict, err := LoadConfig(); err == nil {
		return Options{
			ProfileName: c.GlobalString("profile"),
			Verbose:     c.GlobalBool("verbose"),
			ProfileDict: dict,
		}, nil
	} else {
		return Options{}, err
	}

}

func main() {
	app := cli.NewApp()
	app.Name = "oauthttp"
	app.Version = Version
	app.Usage = "HTTP CLI client with OAuth2 authentication"
	app.Author = "Daisuke Miyamoto"
	app.Email = "miyamoto.daisuke@classmethod.jp"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "profile, P",
			Value:  "default",
			Usage:  "profile name",
			EnvVar: "OAUTHTTP_PROFILE",
		},

		cli.BoolFlag{
			Name:  "insecure, k",
			Usage: "Disable SSL certificate verification",
		},
		cli.BoolFlag{
			Name:  "no-body, B",
			Usage: "Disable the body printing to stdout",
		},
		cli.StringFlag{
			Name:  "print-header, H",
			Usage: "Enable the response header printing to stdout (comma separated names)",
		},
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "Run in Verbose mode (logs to stderr)",
		},
	}
	app.Commands = Commands

	app.Run(os.Args)
}

func Tracef(format string, args ...interface{}) {
	if CurrentOptions.Verbose {
		log.Printf(format, args...)
	}
}
