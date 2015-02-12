package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func AccessToken(profileName string) (string, error) {
	currentProfile := CurrentOptions.ProfileDict[profileName]
	if currentProfile == nil {
		return "", fmt.Errorf("unknown profile [%s]", profileName)
	}
	//	if _, ok := currentProfile[AUTH_SERVER_AUTH_ENDPOINT]; !ok {
	//		return "", fmt.Errorf("%s is required in profile [%s]", AUTH_SERVER_AUTH_ENDPOINT, profileName)
	//	}
	if _, ok := currentProfile[AUTH_SERVER_TOKEN_ENDPOINT]; !ok {
		return "", fmt.Errorf("%s is required in profile [%s]", AUTH_SERVER_TOKEN_ENDPOINT, profileName)
	}

	grantType := currentProfile[GRANT_TYPE]
	if grantType == "" {
		grantType = DEFAULT_GRANT_TYPE
	}
	oauth2Conf := newConf(currentProfile)

	switch grantType {
	case "authorization_code":
		state := random()
		url := oauth2Conf.AuthCodeURL(state)
		fmt.Fprintf(os.Stderr, "Open %s and get code\n", url)

		reader := bufio.NewReader(os.Stdin)
		fmt.Fprint(os.Stderr, "Enter code: ")
		if code, err := reader.ReadString('\n'); err != nil {
			return "", err
		} else if tok, err := oauth2Conf.Exchange(oauth2.NoContext, trimSuffix(code, "\n")); err != nil {
			return "", err
		} else {
			return tok.AccessToken, nil
		}
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
		if sourceToken, err := AccessToken(sourceProfile); err != nil {
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

func newConf(profile map[string]string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     getOrDefault(profile, CLIENT_ID, DEFAULT_CLIENT_ID),
		ClientSecret: getOrDefault(profile, CLIENT_SECRET, DEFAULT_CLIENT_SECRET),
		RedirectURL:  profile[REDIRECT],
		Scopes:       strings.Split(getOrDefault(profile, SCOPES, DEFAULT_SCOPES), ","),
		Endpoint: oauth2.Endpoint{
			AuthURL:  profile[AUTH_SERVER_AUTH_ENDPOINT],
			TokenURL: profile[AUTH_SERVER_TOKEN_ENDPOINT],
		},
	}
}

func random() string {
	var n uint64
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return strconv.FormatUint(n, 36)
}

func trimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func condVal(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func getOrDefault(target map[string]string, key string, defaultValue string) string {
	if value, ok := target[key]; ok {
		return value
	}
	return defaultValue
}

func toExpiry(es ...string) time.Time {
	for _, e := range es {
		expires, err := strconv.Atoi(e)
		if err != nil {
			return time.Now().Add(time.Duration(expires) * time.Second)
		}
	}
	return time.Time{}
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
			Expiry:       toExpiry(vals.Get("expires_in"), vals.Get("expires")),
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
