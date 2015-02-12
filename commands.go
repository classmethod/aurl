package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

var DefaultFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "data",
		Usage: "Body data",
	},
}

var Commands = []cli.Command{
	commandGet,
	commandPost,
	commandPut,
	commandDelete,
}

var commandGet = cli.Command{
	Name:      "get",
	ShortName: "g",
	Usage:     "Make GET request",
	Description: `
`,
	Flags:  DefaultFlags,
	Action: doGet,
}

var commandPost = cli.Command{
	Name:      "post",
	ShortName: "p",
	Usage:     "Make POST request",
	Description: `
`,
	Flags:  DefaultFlags,
	Action: doPost,
}

var commandPut = cli.Command{
	Name:  "put",
	Usage: "Make PUT request",
	Description: `
`,
	Flags:  DefaultFlags,
	Action: doPut,
}

var commandDelete = cli.Command{
	Name:      "delete",
	ShortName: "d",
	Usage:     "Make DELETE request",
	Description: `
`,
	Flags:  DefaultFlags,
	Action: doDelete,
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func doGet(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "GET")
}

func doPost(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "POST")
}

func doPut(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "POST")
}

func doDelete(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "DELETE")
}

func loadOptions(ctx *cli.Context) {
	var err error
	CurrentOptions, err = Opts(ctx)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func doRequest(ctx *cli.Context, method string) {
	Tracef("doRequest start")
	result, err := doRequest0(ctx, method)
	Tracef("request done")
	if result != nil {
		Tracef("result found")
		fmt.Println(string(result))
	} else {
		Tracef("no result")
	}
	if err != nil {
		Tracef("error found")
		log.Fatal(err)
		os.Exit(1)
	} else {
		Tracef("no error")
	}
	Tracef("doRequest end")
}

func doRequest0(ctx *cli.Context, method string) (result []byte, err error) {
	Tracef("profileName = %s", CurrentOptions.ProfileName)

	targetUrl, err := targetUrl(ctx)
	if err != nil {
		return nil, err
	}
	Tracef("targetUrl = %s", targetUrl)
	data := ctx.String("data")
	Tracef("data = %s", data)
	body := strings.NewReader(data)
	req, err := http.NewRequest(method, targetUrl, body)
	if err != nil {
		return nil, err
	}
	at, err := AccessToken(CurrentOptions.ProfileName)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", ctx.App.Name, Version))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at))
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	Tracef("request = %s", string(dump))
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	dumpResp, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Printf("%+v%n", err)
	} else {
		Tracef("response = %s", string(dumpResp))
	}
	return ioutil.ReadAll(resp.Body)
}

func targetUrl(ctx *cli.Context) (string, error) {
	if len(ctx.Args()) < 1 {
		return "", fmt.Errorf("target URL required")
	}
	return ctx.Args()[0], nil
}
