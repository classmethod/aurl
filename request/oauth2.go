package request

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/classmethod/aurl/util"
	"github.com/classmethod/aurl/vault"
	"github.com/toqueteos/webbrowser"
)

// OAuth2TokenResponse represents the token response from OAuth2 authorization server
type OAuth2TokenResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    *int64
	Scope        string
	IdToken      string
}

func authCodeGrant(config *vault.Config, credentials *vault.Credentials, insecure bool) (*OAuth2TokenResponse, error) {
	state, err := random()
	if err != nil {
		return nil, err
	}
	authZRequestUrl := authorizationRequestURL("code", config.AuthorizationEndpoint, credentials.ClientId, config.RedirectURI, config.Scope, state)

	fmt.Fprintf(os.Stderr, "Open browser and get code from %s\n", authZRequestUrl)
	if err := webbrowser.Open(authZRequestUrl); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}

	code, err := util.TerminalPrompt("Enter Grant Type: ")
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {config.RedirectURI},
	}
	return tokenRequest(values, config.TokenEndpoint, credentials.ClientId, credentials.ClientSecret, config.UserAgent, insecure)
}

func implicitGrant(config *vault.Config, credentials *vault.Credentials, insecure bool) (*OAuth2TokenResponse, error) {
	state, err := random()
	if err != nil {
		return nil, err
	}
	authUrl := authorizationRequestURL("token", config.AuthorizationEndpoint, credentials.ClientId, config.RedirectURI, config.Scope, state)
	fmt.Fprintf(os.Stderr, "Open browser and get token from %s\n", authUrl)
	if err := webbrowser.Open(authUrl); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}

	body, err := util.TerminalPrompt("Enter Token: ")
	if err != nil {
		return nil, err
	}

	var token vault.Tokens
	if err = json.Unmarshal([]byte(body), &token); err != nil {
		log.Printf("Failed to parse token response: %v", err)
		return nil, err
	}

	return &OAuth2TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		Scope:        token.Scope,
		IdToken:      token.IdToken,
	}, nil
}

func resourceOwnerPasswordCredentialsGrant(config *vault.Config, credentials *vault.Credentials, insecure bool) (*OAuth2TokenResponse, error) {
	values := url.Values{
		"grant_type": {"password"},
		"username":   {credentials.Username},
		"password":   {credentials.Password},
		"scope":      condVal(strings.Join(strings.Split(config.Scope, ","), " ")),
	}
	return tokenRequest(values, config.TokenEndpoint, credentials.ClientId, credentials.ClientSecret, config.UserAgent, insecure)
}

func clientCredentialsGrant(config *vault.Config, credentials *vault.Credentials, insecure bool) (*OAuth2TokenResponse, error) {
	values := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      condVal(strings.Join(strings.Split(config.Scope, ","), " ")),
	}
	return tokenRequest(values, config.TokenEndpoint, credentials.ClientId, credentials.ClientSecret, config.UserAgent, insecure)
}

func refreshGrant(config *vault.Config, credentials *vault.Credentials, refreshToken string, insecure bool) (*OAuth2TokenResponse, error) {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         condVal(strings.Join(strings.Split(config.Scope, ","), " ")),
	}
	return tokenRequest(values, config.TokenEndpoint, credentials.ClientId, credentials.ClientSecret, config.UserAgent, insecure)
}

func authorizationRequestURL(responseType, authEndpoint, clientId, redirectURI, scope, state string) string {
	var buf bytes.Buffer
	buf.WriteString(authEndpoint)
	v := url.Values{
		"response_type": {responseType},
		"client_id":     {clientId},
		"redirect_uri":  condVal(redirectURI),
		"scope":         condVal(strings.Join(strings.Split(scope, ","), " ")),
		"state":         condVal(state),
	}
	if strings.Contains(authEndpoint, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

func tokenRequest(v url.Values, tokenEndpoint, clientId, clientSecret, userAgent string, insecure bool) (*OAuth2TokenResponse, error) {
	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", userAgent)

	req.SetBasicAuth(clientId, clientSecret)

	if dumpReq, err := httputil.DumpRequestOut(req, true); err == nil {
		log.Printf("Token request >>>\n%s\n<<<", string(dumpReq))
	} else {
		log.Printf("Token request dump failed: %s", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Token request failed: %s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()

	if dumpResp, err := httputil.DumpResponse(resp, true); err == nil {
		log.Printf("Token response >>>\n%s\n<<<", string(dumpResp))
	} else {
		log.Printf("Token response dump failed: %s", err)
	}

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var token vault.Tokens
		if err = json.Unmarshal(body, &token); err != nil {
			log.Printf("Failed to parse token response: %v", err)
			return nil, err
		}

		return &OAuth2TokenResponse{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			ExpiresIn:    token.ExpiresIn,
			Scope:        token.Scope,
			IdToken:      token.IdToken,
		}, nil
	}

	return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
}

func condVal(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func random() (string, error) {
	var n uint64
	if err := binary.Read(rand.Reader, binary.LittleEndian, &n); err != nil {
		return "", err
	}

	return strconv.FormatUint(n, 36), nil
}
