package request

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/toqueteos/webbrowser"
)

func authCodeGrant(request *AurlExecution) (*string, error) {
	state := random()
	authZRequestUrl := authorizationRequestURL("code", request, state)
	webbrowser.Open(authZRequestUrl)
	fmt.Fprintf(os.Stderr, "Open browser and get code from %s\n", authZRequestUrl)

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "Enter code: ")
	if code, _, err := reader.ReadLine(); err != nil {
		return nil, err
	} else {
		values := url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {string(code)},
			"redirect_uri": {request.Profile.RedirectURI},
		}
		return tokenRequest(values, request)
	}
}

func implicitGrant(request *AurlExecution) (*string, error) {
	state := random()
	url := authorizationRequestURL("token", request, state)
	webbrowser.Open(url)
	fmt.Fprintf(os.Stderr, "Open browser and get token from %s\n", url)

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "Enter token: ")
	if token, _, err := reader.ReadLine(); err != nil {
		return nil, err
	} else {
		s := "{\"token_type\": \"bearer\",\"access_token\": \"" + string(token) + "\"}" // TODO
		return &s, nil
	}
}

func resourceOwnerPasswordCredentialsGrant(request *AurlExecution) (*string, error) {
	values := url.Values{
		"grant_type": {"password"},
		"username":   {request.Profile.Username},
		"password":   {request.Profile.Password},
		"scope":      condVal(strings.Join(strings.Split(request.Profile.Scope, ","), " ")),
	}
	return tokenRequest(values, request)
}

func clientCredentialsGrant(request *AurlExecution) (*string, error) {
	values := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      condVal(strings.Join(strings.Split(request.Profile.Scope, ","), " ")),
	}
	return tokenRequest(values, request)
}

func refreshGrant(request *AurlExecution, refreshToken string) (*string, error) {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         condVal(strings.Join(strings.Split(request.Profile.Scope, ","), " ")),
	}
	return tokenRequest(values, request)
}

func authorizationRequestURL(responseType string, request *AurlExecution, state string) string {
	var buf bytes.Buffer
	buf.WriteString(request.Profile.AuthorizationEndpoint)
	v := url.Values{
		"response_type": {responseType},
		"client_id":     {request.Profile.ClientId},
		"redirect_uri":  condVal(request.Profile.RedirectURI),
		"scope":         condVal(strings.Join(strings.Split(request.Profile.Scope, ","), " ")),
		"state":         condVal(state),
	}
	if strings.Contains(request.Profile.AuthorizationEndpoint, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

func tokenRequest(v url.Values, request *AurlExecution) (*string, error) {
	req, err := http.NewRequest("POST", request.Profile.TokenEndpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	if request.Headers.Get("User-Agent") == "" {
		req.Header.Add("User-Agent", fmt.Sprintf("%s-%s", request.Name, request.Version))
	} else {
		req.Header.Add("User-Agent", request.Headers.Get("User-Agent"))
	}
	req.SetBasicAuth(request.Profile.ClientId, request.Profile.ClientSecret)

	if dumpReq, err := httputil.DumpRequestOut(req, true); err == nil {
		log.Printf("Token request >>>\n%s\n<<<", string(dumpReq))
	} else {
		log.Printf("Token request dump failed: %s", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *request.Insecure,
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
		if b, err := ioutil.ReadAll(resp.Body); err == nil {
			s := string(b)
			return &s, nil
		} else {
			return nil, err
		}
	} else {
		log.Printf("Token request failed: %d", resp.StatusCode)
		return nil, err
	}
}

func condVal(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func random() string {
	var n uint64
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return strconv.FormatUint(n, 36)
}
