package request

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/vault"
)

func newRequest(target string, insecure, printBody, printHeaders bool) *Request {
	method, data := "GET", ""
	headers := []string{}
	return &Request{
		Name:      "test",
		Config:    &vault.Config{GrantType: "client_credentials", UserAgent: "aurl-test", ContentType: "application/json"},
		TokenInfo: &vault.TokenInfo{Tokens: &vault.Tokens{AccessToken: "dummy"}},
		Method:    &method, Headers: &headers, Data: &data, TargetUrl: &target,
		Insecure: &insecure, PrintBody: &printBody, PrintHeaders: &printHeaders,
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.Write([]byte("body"))
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		if err := newRequest(srv.URL, false, true, true).Execute(keyring.NewArrayKeyring(nil)); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.SplitN(out, "\n", 2)
	if !strings.Contains(lines[0], `"X-Test":["yes"]`) {
		t.Errorf("first line should be header JSON, got %q", lines[0])
	}
	if lines[1] != "body" {
		t.Errorf("body = %q", lines[1])
	}

	out = captureStdout(t, func() {
		if err := newRequest(srv.URL, false, false, true).Execute(keyring.NewArrayKeyring(nil)); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Contains(out, "body") {
		t.Errorf("--no-print-body should suppress body, got %q", out)
	}
}

func TestValidTokenSkipsCredentials(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotAuth = req.Header.Get("Authorization")
	}))
	defer srv.Close()

	if err := newRequest(srv.URL, false, false, false).Execute(keyring.NewArrayKeyring(nil)); err != nil {
		t.Fatalf("Execute without credentials in keyring: %v", err)
	}
	if gotAuth != "Bearer dummy" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}

func TestRedirect(t *testing.T) {
	var gotAuth string
	var other *httptest.Server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/same":
			http.Redirect(w, req, "/get", http.StatusFound)
		case "/cross":
			http.Redirect(w, req, other.URL+"/get", http.StatusFound)
		default:
			gotAuth = req.Header.Get("Authorization")
			w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()
	other = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t.Error("cross-origin redirect must not be followed")
	}))
	defer other.Close()

	if err := newRequest(srv.URL+"/same", false, false, false).Execute(keyring.NewArrayKeyring(nil)); err != nil {
		t.Fatalf("same-origin redirect: %v", err)
	}
	if gotAuth != "Bearer dummy" {
		t.Errorf("Authorization after same-origin redirect = %q", gotAuth)
	}

	err := newRequest(srv.URL+"/cross", false, false, false).Execute(keyring.NewArrayKeyring(nil))
	if err == nil || !strings.Contains(err.Error(), "non-same origin") {
		t.Errorf("cross-origin redirect: err = %v", err)
	}
}

func TestInsecure(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	err := newRequest(srv.URL, false, false, false).Execute(keyring.NewArrayKeyring(nil))
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Errorf("self-signed without -k: err = %v", err)
	}
	if err := newRequest(srv.URL, true, false, false).Execute(keyring.NewArrayKeyring(nil)); err != nil {
		t.Errorf("self-signed with -k: %v", err)
	}
}

// http.ProxyFromEnvironment reads the env once per process, so the proxied case runs in a child process.
func TestProxy(t *testing.T) {
	if os.Getenv("AURL_TEST_PROXY_CHILD") == "1" {
		// Loopback targets are never proxied, so use a host that only the proxy can "resolve".
		err := newRequest("http://example.invalid/get", false, false, false).Execute(keyring.NewArrayKeyring(nil))
		if err != nil {
			t.Fatalf("via proxy: %v", err)
		}
		return
	}

	proxied := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Host == "example.invalid" {
			proxied = true
		}
		w.Write([]byte("via proxy"))
	}))
	defer proxy.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestProxy$")
	cmd.Env = append(os.Environ(), "AURL_TEST_PROXY_CHILD=1", "HTTP_PROXY="+proxy.URL)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v\n%s", err, out)
	}
	if !proxied {
		t.Error("request did not go through the proxy")
	}

	cmd = exec.Command(os.Args[0], "-test.run=^TestProxy$")
	cmd.Env = append(os.Environ(), "AURL_TEST_PROXY_CHILD=1", "HTTP_PROXY=http://127.0.0.1:9")
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "connection refused") {
		t.Errorf("unreachable proxy should fail with connection refused: err=%v\n%s", err, out)
	}
}
