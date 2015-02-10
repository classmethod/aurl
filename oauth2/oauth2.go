package oauth2

import (
	//	"fmt"
	"github.com/classmethod-aws/oauthttp/profile"
	"golang.org/x/oauth2"
	"log"
	"os"
)

func newConf(url string) *oauth2.Config {
	// TODO customize configs
	return &oauth2.Config{
		ClientID:     profile.DEFAULT_CLIENT_ID,
		ClientSecret: profile.DEFAULT_CLIENT_SECRET,
		RedirectURL:  "REDIRECT_URL",
		Scopes:       []string{"read", "write"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  url + "/auth",
			TokenURL: url + "/token",
		},
	}
}

func GetToken(config map[string]string) *oauth2.Token {
	conf := newConf(config[profile.AUTH_SERVER_ENDPOINT])
	tok, err := conf.PasswordCredentialsToken(oauth2.NoContext, config[profile.USERNAME], config[profile.PASSWORD])
	if err != nil {
		log.Fatal(err)
	}
	//	fmt.Printf("%+v\n", tok)
	return tok
}

func GetToken2(config map[string]string, sourceToken string) *oauth2.Token {
	//	conf := newConf(config[profile.AUTH_SERVER_ENDPOINT])
	//	tok, err := conf.SwithUserToken(oauth2.NoContext, config[profile.USERNAME], sourceToken)
	//	if err != nil {
	//		log.Fatal(err)
	log.Fatal("grant_type [switch_user] is not implemented yet")
	os.Exit(1)
	//	}
	//	//	fmt.Printf("%+v\n", tok)
	//	return tok
	return nil
}

//func (c *oauth2.Config) SwithUserToken(ctx oauth2.Context, username, password string) (*oauth2.Token, error) {
//	return retrieveToken(ctx, c, url.Values{
//		"grant_type":   {"switch_user"},
//		"username":     {config[profile.USERNAME]},
//		"access_token": {sourceToken},
//		"scope":        condVal(strings.Join(c.Scopes, " ")),
//	})
//}
//
//func condVal(v string) []string {
//	if v == "" {
//		return nil
//	}
//	return []string{v}
//}
