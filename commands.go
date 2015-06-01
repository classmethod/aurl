package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
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
	commandHead,
	commandOptions,
	commandPatch,
	commandTrace,
}

var commandGet = cli.Command{
	Name:      "get",
	ShortName: "g",
	Usage:     "Make GET request",
	Description: `
The GET method means retrieve whatever information (in the form of an entity) is identified by the Request-URI.
`,
	Flags:  DefaultFlags,
	Action: doGet,
}

var commandPost = cli.Command{
	Name:      "post",
	ShortName: "p",
	Usage:     "Make POST request",
	Description: `
The POST method is used to request that the origin server accept the entity enclosed in the request
as a new subordinate of the resource identified by the Request-URI in the Request-Line.
`,
	Flags:  DefaultFlags,
	Action: doPost,
}

var commandPut = cli.Command{
	Name:  "put",
	Usage: "Make PUT request",
	Description: `
The PUT method requests that the enclosed entity be stored under the supplied Request-URI.
`,
	Flags:  DefaultFlags,
	Action: doPut,
}

var commandDelete = cli.Command{
	Name:      "delete",
	ShortName: "d",
	Usage:     "Make DELETE request",
	Description: `
The DELETE method requests that the origin server delete the resource identified by the Request-URI.
`,
	Flags:  DefaultFlags,
	Action: doDelete,
}

var commandHead = cli.Command{
	Name:      "head",
	ShortName: "h",
	Usage:     "Make HEAD request",
	Description: `
The HEAD method is identical to GET except that the server MUST NOT return a message-body in the response.
`,
	Flags:  DefaultFlags,
	Action: doHead,
}

var commandOptions = cli.Command{
	Name:      "options",
	ShortName: "o",
	Usage:     "Make OPTIONS request",
	Description: `
The OPTIONS method represents a request for information about the communication options available
on the request/response chain identified by the Request-URI.
`,
	Flags:  DefaultFlags,
	Action: doOptions,
}

var commandPatch = cli.Command{
	Name:  "patch",
	Usage: "Make PATCH request",
	Description: `
The PATCH method requests that a set of changes described in the request entity be applied
to the resource identified by the Request-URI.
`,
	Flags:  DefaultFlags,
	Action: doPatch,
}

var commandTrace = cli.Command{
	Name:      "trace",
	ShortName: "t",
	Usage:     "Make TRACE request",
	Description: `
The TRACE method is used to invoke a remote, application-layer loop-back of the request message.
`,
	Flags:  DefaultFlags,
	Action: doTrace,
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
	doRequest(ctx, "PUT")
}

func doDelete(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "DELETE")
}

func doHead(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "HEAD")
}

func doOptions(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "OPTIONS")
}

func doPatch(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "PATCH")
}

func doTrace(ctx *cli.Context) {
	loadOptions(ctx)
	doRequest(ctx, "TRACE")
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
	var tr = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: ctx.GlobalBool("insecure"),
		},
	}
	http.DefaultClient = &http.Client{Transport: tr}

	Tracef("doRequest start")
	resp, tok, err := doRequest0(ctx, method)
	if tok != nil && resp != nil && resp.StatusCode != 401 && resp.StatusCode != 403 {
		storeToken(tok)
	}
	if err != nil {
		log.Fatal(err)
		return
	} else {
		Tracef("request done successfully")
	}

	Tracef("printing headers")
	if ctx.GlobalBool("print-headers") {
		headers, _ := json.Marshal(resp.Header)
		fmt.Println(string(headers))
	}

	if ctx.GlobalBool("no-body") {
		Tracef("printing body is disabled")
	} else {
		Tracef("printing body")
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Tracef("error on read: %v", err)
		} else if body == nil {
			Tracef("no body")
		} else {
			Tracef("body found")
			fmt.Println(string(body))
		}
	}
	Tracef("doRequest end")
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func doRequest0(ctx *cli.Context, method string) (*http.Response, *oauth2.Token, error) {
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
		for _, element := range ctx.GlobalStringSlice("header") {
			s := regexp.MustCompile(":").Split(element, 2)
			headerKey := strings.TrimSpace(s[0])
			headerValue := strings.TrimSpace(s[1])
			req.Header.Set(headerKey, headerValue)
			Tracef("custom header [%s: %s]", headerKey, headerValue)
		}
		Tracef("header end")

		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			Tracef("phase %s failed (request dump failed)", toString(retrieve))
			lastError = err
			continue
		}
		Tracef("request = %s", string(dump))

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ctx.GlobalBool("insecure"),
			},
		}
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				Tracef("redirect to %s", req.URL.String())
				req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", ctx.App.Name, Version))
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tok.AccessToken))
				for _, element := range ctx.StringSlice("header") {
					s := regexp.MustCompile(":").Split(element, 2)
					req.Header.Set(strings.TrimSpace(s[0]), strings.TrimSpace(s[1]))
				}
				return nil
			},
			Transport: tr,
		}
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
		return resp, tok, err
	}
	return nil, nil, fmt.Errorf("%v", lastError)
}

func toString(retrieve bool) string {
	if retrieve {
		return "retrieve"
	}
	return "stored"
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
