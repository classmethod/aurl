package main

import (
	"fmt"
	"github.com/classmethod-aws/oauthttp/oauth2"
	"github.com/classmethod-aws/oauthttp/profile"
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

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func doGet(ctx *cli.Context) {
	doRequest(ctx, "GET")
}

func doPost(ctx *cli.Context) {
	doRequest(ctx, "POST")
}

func doPut(ctx *cli.Context) {
	doRequest(ctx, "POST")
}

func doDelete(ctx *cli.Context) {
	doRequest(ctx, "DELETE")
}

func doRequest(ctx *cli.Context, method string) {
	apiUrl := getApiUrl(ctx)
	debug("url = %s", apiUrl)
	body := strings.NewReader(ctx.GlobalString("data"))

	profileName := ctx.GlobalString("profile")
	debug("profile = ", profileName)

	req, _ := http.NewRequest(method, apiUrl, body)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getAccessToken(ctx)))

	dump, _ := httputil.DumpRequestOut(req, true)
	debug("request = ", string(dump))

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	dumpResp, _ := httputil.DumpResponse(resp, true)
	debug("response = ", string(dumpResp))

	byteArray, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(byteArray))
}

func getAccessToken(ctx *cli.Context) string {
	profileName := ctx.GlobalString("profile")
	return getAccessToken0(ctx, profileName)
}

func getAccessToken0(ctx *cli.Context, profileName string) string {
	config := profile.ParseConfig()
	if config[profileName] == nil {
		log.Fatal(fmt.Sprintf("unknown profile [%s]", profileName))
		os.Exit(1)
	}

	gt := config[profileName][profile.GRANT_TYPE]
	if gt == "" {
		gt = profile.DEFAULT_GRANT_TYPE
	}
	switch gt {
	case "password":
		tok := oauth2.GetToken(config[profileName])
		return string(tok.AccessToken)
	case "switch_user":
		sourceProfileName := config[profileName][profile.SOURCE_PROFILE]
		t := getAccessToken0(ctx, sourceProfileName)
		tok := oauth2.GetToken2(config[profileName], t)
		return string(tok.AccessToken)
	}
	log.Fatal(fmt.Sprintf("unknown grant_type [%s] in profile [%s]", gt, profileName))
	os.Exit(1)
	return ""
}

func getApiUrl(ctx *cli.Context) string {
	if len(ctx.Args()) < 1 {
		log.Fatal("require URL" + string(len(ctx.Args())))
		os.Exit(1)
	}
	return ctx.Args()[0]
}
