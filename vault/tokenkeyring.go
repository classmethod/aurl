package vault

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/byteness/keyring"
)

const TokenKeyringSuffix = "-TokenInfo"

type Tokens struct {
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    *int64 `json:"expires_in"`
	Expires      *int64 `json:"expires"` // broken Facebook spelling of expires_in
}

type TokenInfo struct {
	Tokens *Tokens
	// RequestTime is the time when the token was requested
	RequestTimestamp int64
}

func (t *TokenInfo) IsExpired() bool {

	if t.Tokens.ExpiresIn == nil {
		return false
	}
	// Consider token expired 60 seconds before actual expiration to avoid edge cases
	expirationTime := t.RequestTimestamp + *t.Tokens.ExpiresIn - 60
	currentTime := time.Now().Unix()
	log.Printf("TokenInfo.IsExpired: CurrentTime=%d, ExpirationTime=%d", currentTime, expirationTime)
	return currentTime >= expirationTime
}

type TokenKeyring struct {
	Keyring keyring.Keyring
}

var ErrNotFound = keyring.ErrKeyNotFound

func (tkr *TokenKeyring) lookupKeyName(key string) (string, error) {
	allKeys, err := tkr.Keyring.Keys()
	if err != nil {
		return key, err
	}
	for _, keyName := range allKeys {
		if strings.EqualFold(keyName, key) {
			return keyName, nil
		}
	}
	return key, ErrNotFound
}

func (tkr *TokenKeyring) Get(profileName string) (tokenInfo *TokenInfo, err error) {
	keyName := profileName + TokenKeyringSuffix
	item, err := tkr.Keyring.Get(keyName)
	if err != nil {
		log.Printf("TokenKeyring.Get: Failed to get token from keyring: %v", err)
		return tokenInfo, err
	}
	if err = json.Unmarshal(item.Data, &tokenInfo); err != nil {
		log.Printf("TokenKeyring: Ignoring invalid data: %s", err.Error())
		return tokenInfo, ErrNotFound
	}
	return tokenInfo, err
}

func (tkr *TokenKeyring) Set(profileName string, tokenInfo *TokenInfo) error {
	keyName := profileName + TokenKeyringSuffix
	valJSON, err := json.Marshal(tokenInfo)
	if err != nil {
		return err
	}

	return tkr.Keyring.Set(keyring.Item{
		Key:         keyName,
		Data:        valJSON,
		Label:       fmt.Sprintf("aurl token for %s (expires in %d seconds)", profileName, tokenInfo.Tokens.ExpiresIn),
		Description: "aurl token",
	})
}

func (tkr *TokenKeyring) Remove(profileName string) error {
	keyName, err := tkr.lookupKeyName(profileName + TokenKeyringSuffix)
	if err != nil && err != ErrNotFound {
		return err
	}

	return tkr.Keyring.Remove(keyName)
}
