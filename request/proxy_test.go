package request

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/classmethod/aurl/vault"
)

// Both the token request and the API request must honor HTTP(S)_PROXY / NO_PROXY,
// the same way http.DefaultTransport does.
func TestRequestsGoThroughProxyFromEnvironment(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		seen = append(seen, req.URL.String())
		mu.Unlock()
		if req.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"x","token_type":"Bearer","expires_in":60}`))
			return
		}
		w.Write([]byte("ok"))
	}))
	defer proxy.Close()

	// http.ProxyFromEnvironment reads the environment only once per process.
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "")

	if _, err := tokenRequest(url.Values{}, "http://auth.example.test/token", "id", "secret", "aurl-test", false); err != nil {
		t.Fatalf("token request: %v", err)
	}

	method, data, target, insecure := "GET", "", "http://api.example.test/resource", false
	headers := []string{}
	r := &Request{
		Config:    &vault.Config{UserAgent: "aurl-test", ContentType: "application/json"},
		TokenInfo: &vault.TokenInfo{Tokens: &vault.Tokens{AccessToken: "x"}},
		Method:    &method, Headers: &headers, Data: &data, TargetUrl: &target, Insecure: &insecure,
	}
	if _, err := r.doRequest(); err != nil {
		t.Fatalf("api request: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{"http://auth.example.test/token", "http://api.example.test/resource"}
	if len(seen) != len(want) || seen[0] != want[0] || seen[1] != want[1] {
		t.Fatalf("requests seen by proxy = %v, want %v", seen, want)
	}
}
