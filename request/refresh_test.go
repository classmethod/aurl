package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/vault"
)

func TestExecuteRefresh(t *testing.T) {
	tests := []struct {
		name             string
		refreshStatus    int
		refreshBody      string
		wantAccessToken  string
		wantRefreshToken string
		wantGrantCalled  bool
	}{
		{
			name:             "refresh fails then falls back to grant",
			refreshStatus:    http.StatusBadRequest,
			refreshBody:      `{"error":"invalid_grant"}`,
			wantAccessToken:  "granted-access",
			wantRefreshToken: "",
			wantGrantCalled:  true,
		},
		{
			name:             "refresh response without refresh_token keeps old one",
			refreshStatus:    http.StatusOK,
			refreshBody:      `{"access_token":"refreshed-access","token_type":"Bearer","expires_in":3600}`,
			wantAccessToken:  "refreshed-access",
			wantRefreshToken: "old-refresh",
			wantGrantCalled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grantCalled := false
			var apiAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/token" {
					apiAuth = req.Header.Get("Authorization")
					w.Write([]byte("ok"))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch req.FormValue("grant_type") {
				case "refresh_token":
					w.WriteHeader(tt.refreshStatus)
					w.Write([]byte(tt.refreshBody))
				case "client_credentials":
					grantCalled = true
					w.Write([]byte(`{"access_token":"granted-access","token_type":"Bearer","expires_in":3600}`))
				default:
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer srv.Close()

			method, data, target := "GET", "", srv.URL+"/resource"
			insecure, printBody, printHeaders := false, false, false
			headers := []string{}
			expiresIn := int64(3600)
			r := &Request{
				Name:        "test",
				Config:      &vault.Config{GrantType: "client_credentials", TokenEndpoint: srv.URL + "/token", UserAgent: "aurl-test", ContentType: "application/json"},
				Credentials: &vault.Credentials{ClientId: "id", ClientSecret: "secret"},
				TokenInfo: &vault.TokenInfo{
					RequestTimestamp: 0,
					Tokens:           &vault.Tokens{AccessToken: "expired-access", RefreshToken: "old-refresh", ExpiresIn: &expiresIn},
				},
				Method: &method, Headers: &headers, Data: &data, TargetUrl: &target,
				Insecure: &insecure, PrintBody: &printBody, PrintHeaders: &printHeaders,
			}

			if err := r.Execute(keyring.NewArrayKeyring(nil)); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if grantCalled != tt.wantGrantCalled {
				t.Errorf("grant called = %v, want %v", grantCalled, tt.wantGrantCalled)
			}
			if want := "Bearer " + tt.wantAccessToken; apiAuth != want {
				t.Errorf("API Authorization = %q, want %q", apiAuth, want)
			}
			if got := r.TokenInfo.Tokens.AccessToken; got != tt.wantAccessToken {
				t.Errorf("access token = %q, want %q", got, tt.wantAccessToken)
			}
			if got := r.TokenInfo.Tokens.RefreshToken; got != tt.wantRefreshToken {
				t.Errorf("refresh token = %q, want %q", got, tt.wantRefreshToken)
			}
		})
	}
}
