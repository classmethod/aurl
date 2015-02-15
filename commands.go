package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"golang.org/x/oauth2"
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
	result, tok, err := doRequest0(ctx, method)
	if err != nil {
		Tracef("error found on request")
		log.Fatal(err)
	} else {
		Tracef("request done successfully")
		if tok != nil {
			storeToken(tok)
		}
	}
	if result != nil {
		Tracef("result found")
		fmt.Println(string(result))
	} else {
		Tracef("no result")
	}
	Tracef("doRequest end")
}

func doRequest0(ctx *cli.Context, method string) ([]byte, *oauth2.Token, error) {
	Tracef("profileName = %s", CurrentOptions.ProfileName)

	targetUrl, err := targetUrl(ctx)
	if err != nil {
		return nil, nil, err
	}
	Tracef("targetUrl = %s", targetUrl)
	data := ctx.String("data")
	Tracef("data = %s", data)
	body := strings.NewReader(data)
	req, err := http.NewRequest(method, targetUrl, body)
	if err != nil {
		return nil, nil, err
	}

	var lastError error
	for retrieve := false; retrieve == false; retrieve = true {
		Tracef("=== phase %s start", toString(retrieve))
		tok, r, err := AccessToken(CurrentOptions.ProfileName, retrieve)
		if err != nil {
			Tracef("phase %s failed (token retrieving failed)", toString(retrieve))
			lastError = err
			continue
		}
		retrieve = r
		req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", ctx.App.Name, Version))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tok.AccessToken))
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			Tracef("phase %s failed (request dump failed)", toString(retrieve))
			lastError = err
			continue
		}
		Tracef("request = %s", string(dump))

		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			Tracef("redirect to %s", req.URL.String())
			req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", ctx.App.Name, Version))
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tok.AccessToken))
			return nil
		}}
		resp, err := client.Do(req)
		if err != nil {
			Tracef("phase %s failed (request failed)", toString(retrieve))
			lastError = err
			continue
		}

		dumpResp, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Printf("%+v%n", err)
		} else {
			Tracef("response = %s", string(dumpResp))
		}

		Tracef("phase %s", toString(retrieve))
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			if retrieve {
				Tracef("phase retrieve failed (4XX response)")
				lastError = err
				break
			} else {
				Tracef("phase stored failed (4XX response) -> final result")
			}
		}

		Tracef("read body %s", toString(retrieve))
		body, err := ioutil.ReadAll(resp.Body)
		return body, tok, err
	}
	return nil, nil, fmt.Errorf("%v", lastError)
}

func toString(retrieve bool) string {
	if retrieve {
		return "retrieve"
	}
	return "sotred"
}

func targetUrl(ctx *cli.Context) (string, error) {
	if len(ctx.Args()) < 1 {
		return "", fmt.Errorf("target URL required")
	}
	return ctx.Args()[0], nil
}

func storeToken(tok *oauth2.Token) {
	if SaveValues(CurrentOptions.ProfileName, tokenToValues(tok)) {
		Tracef("token stored [%s]", CurrentOptions.ProfileName)
	} else {
		Tracef("fail to store token [%s]", CurrentOptions.ProfileName)
	}
}
