package request

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/vault"
)

type Request struct {
	Name    string
	Version string

	Config       *vault.Config
	Credentials  *vault.Credentials
	TokenInfo    *vault.TokenInfo
	Method       *string
	Headers      *[]string
	Data         *string
	Insecure     *bool
	PrintBody    *bool
	PrintHeaders *bool

	TargetUrl *string
}

func (r *Request) Execute(keyring keyring.Keyring) (err error) {
	log.Printf("Profile = %v", r.Config)
	if r.TokenInfo == nil || r.TokenInfo.Tokens.AccessToken == "" || r.TokenInfo.IsExpired() {
		var tokenResponse *OAuth2TokenResponse

		if r.TokenInfo != nil && r.TokenInfo.Tokens != nil && r.TokenInfo.Tokens.RefreshToken != "" {
			log.Printf("Access token is missing or expired, try to refresh using refresh token")
			if tokenResponse, err = r.refresh(); err != nil {
				log.Printf("Token refresh failed: %v", err)
			}
		} else {
			log.Printf("Access token is missing or expired, perform full grant flow")
			if tokenResponse, err = r.grant(); err != nil {
				return err
			}
		}

		log.Printf("Obtained tokens: %v", tokenResponse)
		r.TokenInfo = &vault.TokenInfo{
			RequestTimestamp: time.Now().Unix(),
			Tokens: &vault.Tokens{
				AccessToken:  tokenResponse.AccessToken,
				RefreshToken: tokenResponse.RefreshToken,
				TokenType:    tokenResponse.TokenType,
				ExpiresIn:    tokenResponse.ExpiresIn,
				Scope:        tokenResponse.Scope,
				IdToken:      tokenResponse.IdToken,
			},
		}

		// Save tokens to keyring
		tkr := &vault.TokenKeyring{Keyring: keyring}
		if err := tkr.Set(r.Name, r.TokenInfo); err != nil {
			log.Printf("Failed to save tokens to keyring: %v", err)
		} else {
			log.Printf("Tokens saved to keyring")
		}
	}

	response, err := r.doRequest()
	if err != nil {
		log.Printf("Request failed: %v", err)
		return err
	}

	r.doPrint(response)
	return nil
}

func (r *Request) refresh() (*OAuth2TokenResponse, error) {
	return refreshGrant(r.Config, r.Credentials, r.TokenInfo.Tokens.RefreshToken, *r.Insecure)
}

func (r *Request) grant() (*OAuth2TokenResponse, error) {
	switch r.Config.GrantType {
	case "authorization_code":
		return authCodeGrant(r.Config, r.Credentials, *r.Insecure)
	case "implicit":
		// TODO: not enough checked yet
		return implicitGrant(r.Config, r.Credentials, *r.Insecure)
	case "password":
		// TODO: not enough checked yet
		return resourceOwnerPasswordCredentialsGrant(r.Config, r.Credentials, *r.Insecure)
	case "client_credentials":
		return clientCredentialsGrant(r.Config, r.Credentials, *r.Insecure)
	default:
		return nil, errors.New("Unknown grant type: " + r.Config.GrantType)
	}
}

func (r *Request) doRequest() (*http.Response, error) {
	body := strings.NewReader(*r.Data)
	httpReq, err := http.NewRequest(*r.Method, *r.TargetUrl, body)
	if err != nil {
		return nil, err
	}

	// Initialize Headers from string slice
	httpReq.Header = http.Header{}
	for _, header := range *r.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			httpReq.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", r.Config.UserAgent)
	}

	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", r.Config.ContentType)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.TokenInfo.Tokens.AccessToken))

	if dumpReq, err := httputil.DumpRequestOut(httpReq, true); err == nil {
		log.Printf("Dominant request >>>\n%s\n<<<", string(dumpReq))
	} else {
		log.Printf("Dominant request dump failed: %v", err)
	}

	client := &http.Client{
		CheckRedirect: func(redirectRequest *http.Request, via []*http.Request) error {
			log.Printf("Redirect to %s", redirectRequest.URL.String())
			log.Printf("Original request Host = %s", httpReq.URL.String())
			// Initialize Headers from string slice
			redirectRequest.Header = http.Header{}
			for _, header := range *r.Headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) == 2 {
					redirectRequest.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
			if redirectRequest.Header.Get("User-Agent") == "" {
				redirectRequest.Header.Set("User-Agent", r.Config.UserAgent)
			}
			if matchServer(redirectRequest.URL, httpReq.URL) {
				log.Printf("Propagate authorization header")
				redirectRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.TokenInfo.Tokens.AccessToken))
				return nil
			} else {
				return errors.New("Redirect to non-same origin resource server")
			}
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *r.Insecure,
			},
		},
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Dominant request failed")
		return resp, err
	}
	defer resp.Body.Close()

	if dumpResp, err := httputil.DumpResponse(resp, true); err == nil {
		log.Printf("Dominant response >>>\n%s\n<<<", string(dumpResp))
	} else {
		log.Printf("Dominant response dump failed: %v", err)
	}

	if resp.StatusCode == 401 {
		return resp, errors.New("401 Unauthorized")
	} else {
		return resp, err
	}
}

func (r *Request) doPrint(response *http.Response) {
	if response == nil {
		return
	}
	if *r.PrintHeaders {
		log.Println("Printing headers")
		headers, err := json.Marshal(response.Header)
		if err == nil {
			fmt.Println(string(headers))
		} else {
			log.Println("Header marshaling failed: ", err)
			log.Println("Continue...")
			fmt.Println("{}")
		}
	} else {
		log.Println("No printing headers")
	}

	if *r.PrintBody {
		log.Println("Printing body")
		_, err := io.Copy(os.Stdout, response.Body)
		if err != nil {
			log.Printf("Error on read: %v", err)
			log.Println()
		}
	} else {
		log.Println("No printing body")
	}
}

func matchServer(a *url.URL, b *url.URL) bool {
	if a.Scheme != b.Scheme {
		return false
	}
	if a.Host != b.Host {
		return false
	}
	return true
}
