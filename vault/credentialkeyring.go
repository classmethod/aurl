package vault

import (
	"encoding/json"
	"fmt"

	"github.com/byteness/keyring"
)

type Credentials struct {
	ClientId     string
	ClientSecret string
	Username     string
	Password     string
}

type CredentialKeyring struct {
	Keyring keyring.Keyring
}

func (ck *CredentialKeyring) Get(credentialsName string) (creds *Credentials, err error) {
	item, err := ck.Keyring.Get(credentialsName)
	if err != nil {
		return creds, err
	}
	if err = json.Unmarshal(item.Data, &creds); err != nil {
		return creds, fmt.Errorf("Invalid data in keyring: %v", err)
	}
	return creds, err
}

func (ck *CredentialKeyring) Set(credentialsName string, creds Credentials) error {
	bytes, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	return ck.Keyring.Set(keyring.Item{
		Key:   credentialsName,
		Label: fmt.Sprintf("aurl (%s)", credentialsName),
		Data:  bytes,

		// specific Keychain settings
		KeychainNotTrustApplication: true,
	})
}

func (ck *CredentialKeyring) Remove(credentialsName string) error {
	return ck.Keyring.Remove(credentialsName)
}
