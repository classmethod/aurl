package profiles

import (
	"fmt"
	"errors"
	ini "github.com/rakyll/goini"
	"github.com/classmethod/aurl/utils"
)

type Profile struct {
	Name string
	ClientId string
	ClientSecret string
	AuthorizationEndpoint string
	TokenEndpoint string
	IntrospectionEndpoint string
	RevocationEndpoint string
	RedirectURI string
	GrantType string
	Scope string
	Username string
	Password string
	DefaultContentType string
}

const (
	DEFAULT_CONFIG_FILE = "~/.aurl/profiles"

	CLIENT_ID                  = "client_id"
	CLIENT_SECRET              = "client_secret"
	AUTH_SERVER_AUTH_ENDPOINT  = "auth_server_auth_endpoint"
	AUTH_SERVER_TOKEN_ENDPOINT = "auth_server_token_endpoint"
	INTROSPECTION_ENDPOINT     = "introspection_endpoint"
	REVOCATION_ENDPOINT        = "revocation_endpoint"
	REDIRECT                   = "redirect"
	GRANT_TYPE                 = "grant_type"
	SCOPES                     = "scopes"
	USERNAME                   = "username"
	PASSWORD                   = "password"
	DEFAULT_CONTENT_TYPE       = "default_content_type"
	//SOURCE_PROFILE             = "source_profile"

	DEFAULT_CLIENT_ID     = "aurl"
	DEFAULT_CLIENT_SECRET = "aurl"
	DEFAULT_GRANT_TYPE    = "authorization_code"
	DEFAULT_SCOPES        = "root"
)

func LoadProfile(profileName string) (Profile, error) {
	if dict, err := loadConfig(); err != nil {
		return Profile{}, err
	} else if p, ok := dict[profileName]; ok {
		return Profile{
			Name:					profileName,
			ClientId:				getOrDefault(p, CLIENT_ID, DEFAULT_CLIENT_ID),
			ClientSecret:			getOrDefault(p, CLIENT_SECRET, DEFAULT_CLIENT_SECRET),
			AuthorizationEndpoint:	getOrDefault(p, AUTH_SERVER_AUTH_ENDPOINT, ""),
			TokenEndpoint:			getOrDefault(p, AUTH_SERVER_TOKEN_ENDPOINT, ""),
			IntrospectionEndpoint:	getOrDefault(p, INTROSPECTION_ENDPOINT, ""),
			RevocationEndpoint:		getOrDefault(p, REVOCATION_ENDPOINT, ""),
			RedirectURI:			getOrDefault(p, REDIRECT, ""),
			GrantType:				getOrDefault(p, GRANT_TYPE, DEFAULT_GRANT_TYPE),
			Scope:					getOrDefault(p, SCOPES, DEFAULT_SCOPES),
			Username:				getOrDefault(p, USERNAME, ""),
			Password:				getOrDefault(p, PASSWORD, ""),
			DefaultContentType:		getOrDefault(p, DEFAULT_CONTENT_TYPE, ""),
		}, nil
	} else {
		return Profile{}, errors.New("Unknown profile: " + profileName)
	}
}

func loadConfig() (map[string]map[string]string, error) {
	return ini.Load(utils.ExpandPath(DEFAULT_CONFIG_FILE))
}

func (p Profile) String() string {
	return fmt.Sprintf("{name:%s, clientId:%s, authEndpoint:%s, tokendEndpoint:%s, redirect:%s, grantType:%s, scooe:%s}",
		p.Name, p.ClientId, p.AuthorizationEndpoint, p.TokenEndpoint, p.RedirectURI, p.GrantType, p.Scope)
}


func getOrDefault(dict map[string]string, key string, defaultValue string) string {
	if v, ok := dict[key]; ok {
		return v
	}
	return defaultValue
}