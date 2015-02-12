package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func AccessToken(profileName string) (string, error) {
	config, _ := LoadConfig()
	return accessToken(config, profileName)
}

func accessToken(config map[string]map[string]string, profileName string) (string, error) {
	currentProfile := config[profileName]
	if currentProfile == nil {
		return "", fmt.Errorf("unknown profile [%s]", profileName)
	}

	grantType := currentProfile[GRANT_TYPE]
	if grantType == "" {
		grantType = DEFAULT_GRANT_TYPE
	}
	oauth2Conf := newConf(currentProfile[AUTH_SERVER_ENDPOINT])

	switch grantType {
	//	case "authorizaton_code":
	//		tok := oauth2.GetToken(currentProfile)
	//		return string(tok.AccessToken)
	case "password":
		username := currentProfile[USERNAME]
		password := currentProfile[PASSWORD]
		if tok, err := oauth2Conf.PasswordCredentialsToken(oauth2.NoContext, username, password); err != nil {
			return "", err
		} else {
			return tok.AccessToken, nil
		}
	case "switch_user":
		username := currentProfile[USERNAME]
		sourceProfile := currentProfile[SOURCE_PROFILE]
		if sourceToken, err := accessToken(config, sourceProfile); err != nil {
			return "", err
		} else if tok, err := retrieveToken(oauth2.NoContext, oauth2Conf, switchUserValues(username, sourceToken, oauth2Conf.Scopes)); err != nil {
			return "", err
		} else {
			return tok.AccessToken, nil
		}
	}
	return "", fmt.Errorf("unknown grant_type [%s] in profile [%s]", grantType, profileName)
}

func switchUserValues(username string, sourceToken string, scopes []string) url.Values {
	return url.Values{
		"grant_type":   {"switch_user"},
		"username":     {username},
		"access_token": {sourceToken},
		"scope":        condVal(strings.Join(scopes, " "))}
}

func newConf(url string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     DEFAULT_CLIENT_ID,
		ClientSecret: DEFAULT_CLIENT_SECRET,
		RedirectURL:  "REDIRECT_URL",
		Scopes:       []string{"read", "write"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  url + "/authorize",
			TokenURL: url + "/token",
		},
	}
}

func condVal(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func retrieveToken(ctx oauth2.Context, conf *oauth2.Config, values url.Values) (*oauth2.Token, error) {
	values.Set("client_id", conf.ClientID)
	req, err := http.NewRequest("POST", conf.Endpoint.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(conf.ClientID, conf.ClientSecret)
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("oauth2: cannot fetch token: %v", err)
	}
	if code := res.StatusCode; code < 200 || code > 299 {
		return nil, fmt.Errorf("oauth2: cannot fetch token: %v\nResponse: %s", res.Status, body)
	}

	var token *oauth2.Token
	content, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	switch content {
	case "application/x-www-form-urlencoded", "text/plain":
		vals, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		token = &oauth2.Token{
			AccessToken:  vals.Get("access_token"),
			TokenType:    vals.Get("token_type"),
			RefreshToken: vals.Get("refresh_token"),
		}
		e := vals.Get("expires_in")
		if e == "" {
			e = vals.Get("expires")
		}
		expires, _ := strconv.Atoi(e)
		if expires != 0 {
			token.Expiry = time.Now().Add(time.Duration(expires) * time.Second)
		}
	default:
		var tj tokenJSON
		if err = json.Unmarshal(body, &tj); err != nil {
			return nil, err
		}
		token = &oauth2.Token{
			AccessToken:  tj.AccessToken,
			TokenType:    tj.TokenType,
			RefreshToken: tj.RefreshToken,
			Expiry:       tj.expiry(),
		}
	}
	// Don't overwrite `RefreshToken` with an empty value
	// if this was a token refreshing request.
	if token.RefreshToken == "" {
		token.RefreshToken = values.Get("refresh_token")
	}
	return token, nil
}

// tokenJSON is the struct representing the HTTP response from OAuth2
// providers returning a token in JSON form.
type tokenJSON struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
	Expires      int32  `json:"expires"` // broken Facebook spelling of expires_in
}

func (e *tokenJSON) expiry() (t time.Time) {
	if v := e.ExpiresIn; v != 0 {
		return time.Now().Add(time.Duration(v) * time.Second)
	}
	if v := e.Expires; v != 0 {
		return time.Now().Add(time.Duration(v) * time.Second)
	}
	return
}
