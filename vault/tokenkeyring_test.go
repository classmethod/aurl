package vault

import (
	"testing"

	"github.com/byteness/keyring"
)

func TestTokenKeyringRemoveMissingIsNoop(t *testing.T) {
	tkr := &TokenKeyring{Keyring: keyring.NewArrayKeyring(nil)}
	if err := tkr.Remove("nope"); err != nil {
		t.Fatalf("Remove of missing token: %v", err)
	}
	if err := tkr.Set("p", &TokenInfo{Tokens: &Tokens{}}); err != nil {
		t.Fatal(err)
	}
	if err := tkr.Remove("p"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := tkr.Get("p"); err == nil {
		t.Fatal("token still present after Remove")
	}
}
