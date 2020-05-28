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

	"github.com/classmethod/aurl/profiles"
	"github.com/classmethod/aurl/tokens"
)

type AurlExecution struct {
	Name    string
	Version string

	Profile      profiles.Profile
	Method       *string
	Headers      *http.Header
	Data         *string
	Insecure     *bool
	PrintBody    *bool
	PrintHeaders *bool

	TargetUrl *string
}

func (execution *AurlExecution) Execute() error {
	log.Printf("Profile = %v", execution.Profile)

	if tokenResponseString, err := tokens.LoadTokenResponseString(execution.Profile.Name); err == nil {
		log.Printf("Stored token response was found: >>>\n%v\n<<<", *tokenResponseString)
		if tokenResponse, err := tokens.New(tokenResponseString); err == nil {
			log.Printf("Stored access token: %v", tokenResponse.AccessToken)
			if tokenResponse.IsExpired() == false {
				response, err := execution.doRequest(tokenResponse, execution.Profile)
				if err == nil {
					log.Printf("Stored access token was valid")
					execution.doPrint(response)
					return nil
				} else {
					log.Printf("Stored access token was invalid: %v", err.Error())
				}
			} else {
				log.Printf("Stored access token was expired")
			}
			if tokenResponse.RefreshToken != "" {
				log.Printf("Stored refresh token: %v", tokenResponse.RefreshToken)
				if tokenResponseString, err = execution.refresh(tokenResponse); err == nil {
					if tokenResponseString != nil {
						log.Printf("Refreshed token response: >>>\n%v\n<<<", *tokenResponseString)
					}
					if tokenResponse, err := tokens.New(tokenResponseString); err == nil {
						if response, err := execution.doRequest(tokenResponse, execution.Profile); err == nil {
							log.Printf("Refreshed access token was valid")
							execution.doPrint(response)
							tokens.SaveTokenResponseString(execution.Profile.Name, tokenResponseString)
							return nil
						} else {
							log.Printf("Refreshed access token was invalid: %v", err.Error())
						}
					} else {
						log.Printf("Failed to parse refreshed token response: %v", err.Error())
					}
				} else {
					log.Printf("Stored refresh token was invalid: %v", err.Error())
				}
			} else {
				log.Printf("Stored refresh token was not found")
			}
		} else {
			log.Printf("Failed to parse stored token response: %v", err.Error())
		}
	} else {
		log.Printf("Stored access token was not found: %v", err.Error())
	}

	if tokenResponseString, err := execution.grant(); err == nil {
		if tokenResponseString != nil {
			log.Printf("Issued token response: >>>\n%v\n<<<", *tokenResponseString)
		}
		if tokenResponse, err := tokens.New(tokenResponseString); err == nil {
			log.Printf("Issued access token: %v", tokenResponse.AccessToken)
			if response, err := execution.doRequest(tokenResponse, execution.Profile); err == nil {
				log.Printf("Issued access token was valid")
				execution.doPrint(response)
				tokens.SaveTokenResponseString(execution.Profile.Name, tokenResponseString)
				return nil
			} else {
				log.Printf("Granted access token was invalid: %v", err.Error())
				return err
			}
		} else {
			log.Printf("Failed to parse granted token response: %v", err.Error())
			return err
		}
	} else {
		log.Printf("Grant failed: %v", err.Error())
		return err
	}
}

func (execution *AurlExecution) refresh(tokenResponse tokens.TokenResponse) (*string, error) {
	return refreshGrant(execution, tokenResponse.RefreshToken)
}

func (execution *AurlExecution) grant() (*string, error) {
	switch execution.Profile.GrantType {
	case "authorization_code":
		return authCodeGrant(execution)
	case "implicit":
		return implicitGrant(execution)
	case "password":
		return resourceOwnerPasswordCredentialsGrant(execution)
	case "client_credentials":
		return clientCredentialsGrant(execution)
	default:
		return nil, errors.New("Unknown grant type: " + execution.Profile.GrantType)
	}
}

func (execution *AurlExecution) doRequest(tokenResponse tokens.TokenResponse, profile profiles.Profile) (*http.Response, error) {
	body := strings.NewReader(*execution.Data)
	req, err := http.NewRequest(*execution.Method, *execution.TargetUrl, body)
	if err != nil {
		return nil, err
	}

	req.Header = *execution.Headers
	if req.Header.Get("User-Agent") == "" {
		if execution.Profile.UserAgent != "" {
			req.Header.Set("User-Agent", execution.Profile.UserAgent)
		} else {
			req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", execution.Name, execution.Version))
		}
	}

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", profile.DefaultContentType)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenResponse.AccessToken))

	if dumpReq, err := httputil.DumpRequestOut(req, true); err == nil {
		log.Printf("Dominant request >>>\n%s\n<<<", string(dumpReq))
	} else {
		log.Printf("Dominant request dump failed: %v", err)
	}

	client := &http.Client{
		CheckRedirect: func(redirectRequest *http.Request, via []*http.Request) error {
			log.Printf("Redirect to %s", redirectRequest.URL.String())
			log.Printf("Original request Host = %s", req.URL.String())
			redirectRequest.Header = *execution.Headers
			if redirectRequest.Header.Get("User-Agent") == "" {
				if execution.Profile.UserAgent != "" {
					req.Header.Set("User-Agent", execution.Profile.UserAgent)
				} else {
					req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", execution.Name, execution.Version))
				}
			}
			if matchServer(redirectRequest.URL, req.URL) {
				log.Printf("Propagate authorization header")
				redirectRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenResponse.AccessToken))
				return nil
			} else {
				return errors.New("Redirect to non-same origin resource server")
			}
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *execution.Insecure,
			},
		},
	}
	resp, err := client.Do(req)
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

func (execution *AurlExecution) doPrint(response *http.Response) {
	if response == nil {
		return
	}
	if *execution.PrintHeaders {
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

	if *execution.PrintBody {
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
