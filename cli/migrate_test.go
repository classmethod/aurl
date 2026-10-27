package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/vault"
)

func TestConvertV1Profile(t *testing.T) {
	tests := []struct {
		name        string
		keys        map[string]string
		wantSection vault.ProfileSection
		wantCreds   vault.Credentials
		wantUnknown []string
	}{
		{
			name:        "defaults",
			keys:        map[string]string{},
			wantSection: vault.ProfileSection{Name: "p", GrantType: "authorization_code", Scope: "root"},
			wantCreds:   vault.Credentials{ClientId: "aurl", ClientSecret: "aurl"},
		},
		{
			name: "renamed keys",
			keys: map[string]string{
				"auth_server_auth_endpoint":  "https://auth/authorize",
				"auth_server_token_endpoint": "https://auth/token",
				"redirect":                   "https://cb",
				"scopes":                     "read write",
				"default_content_type":       "text/plain",
				"default_user_agent":         "ua",
				"client_id":                  "id",
				"client_secret":              "secret",
			},
			wantSection: vault.ProfileSection{
				Name:                    "p",
				GrantType:               "authorization_code",
				AuthServerAuthEndpoint:  "https://auth/authorize",
				AuthServerTokenEndpoint: "https://auth/token",
				Redirect:                "https://cb",
				Scope:                   "read write",
				ContentType:             "text/plain",
				UserAgent:               "ua",
			},
			wantCreds: vault.Credentials{ClientId: "id", ClientSecret: "secret"},
		},
		{
			name:        "password grant",
			keys:        map[string]string{"grant_type": "password", "username": "u", "password": "pw", "scopes": ""},
			wantSection: vault.ProfileSection{Name: "p", GrantType: "password"},
			wantCreds:   vault.Credentials{ClientId: "aurl", ClientSecret: "aurl", Username: "u", Password: "pw"},
		},
		{
			name:        "unknown keys",
			keys:        map[string]string{"source_profile": "x", "foo": "y"},
			wantSection: vault.ProfileSection{Name: "p", GrantType: "authorization_code", Scope: "root"},
			wantCreds:   vault.Credentials{ClientId: "aurl", ClientSecret: "aurl"},
			wantUnknown: []string{"foo", "source_profile"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section, creds, unknown := convertV1Profile("p", tt.keys)
			if section != tt.wantSection {
				t.Errorf("section = %+v, want %+v", section, tt.wantSection)
			}
			if creds != tt.wantCreds {
				t.Errorf("creds = %+v, want %+v", creds, tt.wantCreds)
			}
			if !reflect.DeepEqual(unknown, tt.wantUnknown) {
				t.Errorf("unknown = %v, want %v", unknown, tt.wantUnknown)
			}
		})
	}
}

func TestIsV1Style(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"-p", "foo", "https://x"}, true},
		{[]string{"-pfoo", "https://x"}, true},
		{[]string{"--profile=foo", "https://x"}, true},
		{[]string{"--profile", "foo", "http://x"}, true},
		{[]string{"https://x"}, false},
		{[]string{"dev", "https://x"}, false},
		{[]string{"dev", "--print-body", "https://x"}, false},
		{[]string{"curl", "foo", "https://x"}, false},
		{[]string{"add", "foo"}, false},
		{[]string{"migrate"}, false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := IsV1Style(tt.args); got != tt.want {
			t.Errorf("IsV1Style(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMigrateCommand(t *testing.T) {
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	v1 := "[default]\nclient_id = id1\nclient_secret = s1\nauth_server_token_endpoint = https://auth/token\ngrant_type = client_credentials\ndefault_content_type = text/plain\n\n" +
		"[dev]\ngrant_type = password\nusername = u\npassword = pw\n\n[empty]\n\n[curl]\nclient_id = c\n"
	if err := os.WriteFile(profiles, []byte(v1), 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config")
	if err := os.WriteFile(configPath, []byte("[dev]\ngrant_type = authorization_code\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AURL_CONFIG_FILE", configPath)
	config, err := vault.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	kr := keyring.NewArrayKeyring(nil)

	if err := MigrateCommand(MigrateCommandInput{ProfilesFile: profiles}, kr, config); err != nil {
		t.Fatal(err)
	}

	reloaded, err := vault.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := reloaded.ProfileNames(), []string{"dev", "default", "empty"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ProfileNames() = %v, want %v", got, want)
	}
	def, _ := reloaded.ProfileSection("default")
	wantDef := vault.ProfileSection{Name: "default", GrantType: "client_credentials", AuthServerTokenEndpoint: "https://auth/token", Scope: "root", ContentType: "text/plain"}
	if def != wantDef {
		t.Errorf("default section = %+v, want %+v", def, wantDef)
	}
	ckr := &vault.CredentialKeyring{Keyring: kr}
	creds, err := ckr.Get("default")
	if err != nil {
		t.Fatal(err)
	}
	if want := (vault.Credentials{ClientId: "id1", ClientSecret: "s1"}); *creds != want {
		t.Errorf("default creds = %+v, want %+v", *creds, want)
	}
	if empty, _ := reloaded.ProfileSection("empty"); empty != (vault.ProfileSection{Name: "empty", GrantType: "authorization_code", Scope: "root"}) {
		t.Errorf("empty section = %+v, want v1 defaults", empty)
	}
	if dev, _ := reloaded.ProfileSection("dev"); dev.GrantType != "authorization_code" {
		t.Errorf("existing dev profile was overwritten without --force: %+v", dev)
	}
	if _, err := ckr.Get("dev"); err == nil {
		t.Error("credentials for skipped dev profile were stored")
	}
	if _, err := ckr.Get("curl"); err == nil {
		t.Error("credentials for reserved curl profile were stored")
	}

	if err := MigrateCommand(MigrateCommandInput{ProfilesFile: profiles, Force: true}, kr, reloaded); err != nil {
		t.Fatal(err)
	}
	if dev, _ := reloaded.ProfileSection("dev"); dev.GrantType != "password" {
		t.Errorf("dev profile not overwritten with --force: %+v", dev)
	}
	if creds, err := ckr.Get("dev"); err != nil || creds.Password != "pw" {
		t.Errorf("dev creds = %+v, %v", creds, err)
	}
}

func TestMigrateCommandV1Parsing(t *testing.T) {
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	v1 := "client_id = top\n[Prod]\nClient_Secret = \"pa#ss\"\nclient_id = 'a;b'\n"
	if err := os.WriteFile(profiles, []byte(v1), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AURL_CONFIG_FILE", filepath.Join(dir, "config"))
	config, err := vault.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	kr := keyring.NewArrayKeyring(nil)

	if err := MigrateCommand(MigrateCommandInput{ProfilesFile: profiles}, kr, config); err != nil {
		t.Fatal(err)
	}

	if got, want := config.ProfileNames(), []string{"prod"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ProfileNames() = %v, want %v", got, want)
	}
	creds, err := (&vault.CredentialKeyring{Keyring: kr}).Get("prod")
	if err != nil {
		t.Fatal(err)
	}
	if want := (vault.Credentials{ClientId: "a;b", ClientSecret: "pa#ss"}); *creds != want {
		t.Errorf("prod creds = %+v, want %+v", *creds, want)
	}
}

func TestMigrateCommandMissingFile(t *testing.T) {
	err := MigrateCommand(MigrateCommandInput{ProfilesFile: filepath.Join(t.TempDir(), "profiles")}, keyring.NewArrayKeyring(nil), &vault.ConfigFile{})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found error", err)
	}
}

func TestExecCommandSuggestsMigrate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AURL_CONFIG_FILE", filepath.Join(home, "config"))
	config, err := vault.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".aurl"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".aurl", "profiles"), []byte("[V1only]\n"), 0600); err != nil {
		t.Fatal(err)
	}

	err = ExecCommand(ExecCommandInput{ProfileName: "missing"}, keyring.NewArrayKeyring(nil), config)
	if err == nil || strings.Contains(err.Error(), "aurl migrate") {
		t.Fatalf("profile not in v1 file: err = %v", err)
	}

	err = ExecCommand(ExecCommandInput{ProfileName: "v1only"}, keyring.NewArrayKeyring(nil), config)
	if err == nil || !strings.Contains(err.Error(), "aurl migrate") {
		t.Fatalf("profile in v1 file: err = %v, want hint to run aurl migrate", err)
	}
}

func TestSelectProfileNoProfilesHint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := selectProfile(nil, "x"); err == nil || !strings.Contains(err.Error(), "aurl add") {
		t.Fatalf("no v1 file: err = %v, want hint to run aurl add", err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".aurl"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".aurl", "profiles"), []byte("[default]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := selectProfile(nil, "x"); err == nil || !strings.Contains(err.Error(), "aurl migrate") {
		t.Fatalf("v1 file present: err = %v, want hint to run aurl migrate", err)
	}
}
