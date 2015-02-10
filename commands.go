package main

import (
	"log"
	"os"

	"github.com/codegangsta/cli"
)

var Commands = []cli.Command{
	commandGet,
	commandPost,
	commandPut,
	commandDelete,
}

var commandGet = cli.Command{
	Name:  "get",
	Usage: "",
	Description: `
`,
	Action: doGet,
}

var commandPost = cli.Command{
	Name:  "post",
	Usage: "",
	Description: `
`,
	Action: doPost,
}

var commandPut = cli.Command{
	Name:  "put",
	Usage: "",
	Description: `
`,
	Action: doPut,
}

var commandDelete = cli.Command{
	Name:  "delete",
	Usage: "",
	Description: `
`,
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

func doGet(c *cli.Context) {
}

func doPost(c *cli.Context) {
}

func doPut(c *cli.Context) {
}

func doDelete(c *cli.Context) {
}
