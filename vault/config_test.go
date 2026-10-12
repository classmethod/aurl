package vault

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProfileNamesIncludesAllProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	content := "[default]\ngrant_type = client_credentials\n\n[dev]\ngrant_type = authorization_code\n\n[empty]\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AURL_CONFIG_FILE", path)

	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.ProfileNames(), []string{"default", "dev"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ProfileNames() = %v, want %v", got, want)
	}
}
