package main

import (
	ini "github.com/rakyll/goini"
)

const (
	DEFAULT_CONFIG_FILE = "~/.oauthttp/profiles"

	CLIENT_ID                  = "client_id"
	CLIENT_SECRET              = "client_secret"
	AUTH_SERVER_AUTH_ENDPOINT  = "auth_server_auth_endpoint"
	AUTH_SERVER_TOKEN_ENDPOINT = "auth_server_token_endpoint"
	REDIRECT                   = "redirect"
	GRANT_TYPE                 = "grant_type"
	SCOPES                     = "scopes"
	USERNAME                   = "username"
	PASSWORD                   = "password"
	SOURCE_PROFILE             = "source_profile"

	DEFAULT_CLIENT_ID     = "oauthttp"
	DEFAULT_CLIENT_SECRET = "oauthttp"
	DEFAULT_GRANT_TYPE    = "authorization_code"
	DEFAULT_SCOPES        = "read,write"
)

func LoadConfig() (map[string]map[string]string, error) {
	return ini.Load(expandPath(DEFAULT_CONFIG_FILE))
}
