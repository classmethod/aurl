package request

import (
	"io"
	"os"
	"fmt"
	"log"
	"errors"
	"strings"
	"net/url"
	"net/http"
	"net/http/httputil"
	"crypto/tls"
	"encoding/json"
	"github.com/classmethod/aurl/profiles"
	"github.com/classmethod/aurl/tokens"
)

type AurlExecution struct {
	Name string
	Version string

	Profile profiles.Profile
	Method *string
	Headers *http.Header
	Data *string
	Insecure *bool
	PrintBody *bool
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
				execution.doPrint(response)
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

func (request *AurlExecution) refresh(tokenResponse tokens.TokenResponse) (*string, error) {
	return refreshGrant(request, tokenResponse.RefreshToken)
}

func (request *AurlExecution) grant() (*string, error) {
	switch request.Profile.GrantType {
	case "authorization_code":	return authCodeGrant(request)
	case "implicit":			return implicitGrant(request)
	case "password":			return resourceOwnerPasswordCredentialsGrant(request)
	case "client_credentials":	return clientCredentialsGrant(request)
	default:					return nil, errors.New("Unknown grant type: " + request.Profile.GrantType)
	}
}

func (request *AurlExecution) doRequest(tokenResponse tokens.TokenResponse, profile profiles.Profile) (*http.Response, error) {
	body := strings.NewReader(*request.Data)
	req, err := http.NewRequest(*request.Method, *request.TargetUrl, body)
	if err != nil {
		return nil, err
	}

	req.Header = *request.Headers
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", fmt.Sprintf("%s-%s", request.Name, request.Version))
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", profile.DefaultContentType)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenResponse.AccessToken))

	if dumpReq, err := httputil.DumpRequestOut(req, true); err == nil {
		log.Printf("Dominant request >>>\n%s\n<<<", string(dumpReq))
	} else {
		log.Printf("Dominant request dump failed: ", err)
	}

	client := &http.Client{
		CheckRedirect: func(redirectRequest *http.Request, via []*http.Request) error {
			log.Printf("Redirect to %s", redirectRequest.URL.String())
			log.Printf("Original request Host = %s", req.URL.String())
			redirectRequest.Header = *request.Headers
			if redirectRequest.Header.Get("User-Agent") == "" {
				redirectRequest.Header.Set("User-Agent", fmt.Sprintf("%s-%s", request.Name, request.Version))
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
				InsecureSkipVerify: *request.Insecure,
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
		log.Printf("Dominant response dump failed: ", err)
	}

	if resp.StatusCode == 401 {
		return resp, errors.New("401 Unauthorized")
	} else {
		return resp, err
	}
}

func (request *AurlExecution) doPrint(response *http.Response) {
	if response == nil {
		return
	}
	if *request.PrintHeaders {
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

	if *request.PrintBody {
		log.Println("Printing body")
		_, err := io.Copy(os.Stdout, response.Body)
		if err != nil {
			log.Println("Error on read: %v", err)
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
