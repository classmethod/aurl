package main

import (
	ini "github.com/rakyll/goini"
	"os/user"
	"strings"
)

const (
	DEFAULT_CONFIG_FILE = "~/.oauthttp/profiles"

	DEFAULT_GRANT_TYPE    = "authorization_code"
	DEFAULT_CLIENT_ID     = "oauthttp"
	DEFAULT_CLIENT_SECRET = "oauthttp"

	AUTH_SERVER_ENDPOINT = "auth_server_endpoint"
	CLIENT_ID            = "client_id"
	CLIENT_SECRET        = "client_secret"
	GRANT_TYPE           = "grant_type"
	USERNAME             = "username"
	PASSWORD             = "password"
	SOURCE_PROFILE       = "source_profile"
)

func LoadConfig() (map[string]map[string]string, error) {
	return ini.Load(configFilePath())
}

func configFilePath() string {
	path := DEFAULT_CONFIG_FILE
	usr, _ := user.Current()
	var dir string = usr.HomeDir
	if last := len(dir) - 1; last >= 0 && dir[last] != '/' {
		dir = dir + "/"
	}
	// Check in case of paths like "/something/~/something/"
	if path[:2] == "~/" {
		path = strings.Replace(path, "~/", dir, 1)
	}
	//	log.Printf("path = %s", path)
	return path
}
