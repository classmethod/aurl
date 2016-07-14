package main

import (
	"log"
	"io"
	"io/ioutil"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/classmethod/aurl/request"
	"github.com/classmethod/aurl/profiles"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

var (
	profileName		= kingpin.Flag("profile", "Set profile name. (default: \"default\")").Short('p').Default("default").String()
	method			= kingpin.Flag("request", "Set HTTP request method. (default: \"GET\")").Short('X').Default("GET").String()
	headers			= HTTPHeader(kingpin.Flag("header", "Add HTTP headers to the request.").Short('H').PlaceHolder("HEADER:VALUE"))
	data			= kingpin.Flag("data", "Set HTTP request body.").Short('d').String()
	insecure		= kingpin.Flag("insecure", "Disable SSL certificate verification.").Short('k').Bool()
	printBody		= kingpin.Flag("print-body", "Enable printing response body to stdout. (default: enabled, try --no-print-body)").Default("true").Bool()
	printHeaders	= kingpin.Flag("print-headers", "Enable printing response headers JSON to stdout. (default: disabled, try --no-print-headers)").Bool()
	verbose			= kingpin.Flag("verbose", "Enable verbose logging to stderr.").Short('v').Bool()

	targetUrl		= kingpin.Arg("url", "The URL to request").Required().String()
)



// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(Version).Author(Author)
	kingpin.CommandLine.GetFlag("version").Short('V')
	kingpin.CommandLine.GetFlag("help").Short('h')
	kingpin.CommandLine.Help = "Command line utility to make HTTP request with OAuth2."
	kingpin.Parse()

	if *verbose {
		log.SetOutput(cli.errStream)
		log.SetPrefix("**** ")
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	log.Println("Parsed arguments:")
	log.Printf("  profile: %s\n", *profileName)
	log.Printf("  request: %s\n", *method)
	log.Printf("  headers: %s\n", *headers)
	log.Printf("  data: %s\n", *data)
	log.Printf("  insecure: %v\n", *insecure)
	log.Printf("  printBody: %v\n", *printBody)
	log.Printf("  printHeaders: %v\n", *printHeaders)
	log.Printf("  verbose: %v\n", *verbose)
	log.Printf("  targetUrl: %v\n", *targetUrl)

	if profile, err := profiles.LoadProfile(*profileName); err != nil {
		kingpin.FatalIfError(err, "Load profile failed")
		return ExitCodeError
	} else {
		execution := &request.AurlExecution{
			Name: Name,
			Version: Version,

			Profile: profile,
			Method: method,
			Headers: headers,
			Data: data,
			Insecure: insecure,
			PrintBody: printBody,
			PrintHeaders: printHeaders,

			TargetUrl: targetUrl,
		}
		kingpin.FatalIfError(execution.Execute(), "Request failed")
		return ExitCodeOK
	}
}
