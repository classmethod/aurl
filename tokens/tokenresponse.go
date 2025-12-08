package tokens

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const KEYRING_SERVICE_NAME = "aurl"

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
	Expires      int32  `json:"expires"` // broken Facebook spelling of expires_in
	Timestamp    int64  `json:"timestamp"`
}

func (tokenResponse TokenResponse) IsExpired() bool {
	return false // TODO
}

func New(tokenResponseString *string) (TokenResponse, error) {
	if tokenResponseString == nil {
		return TokenResponse{}, errors.New("nil response")
	}
	var tokenResponse TokenResponse
	jsonParser := json.NewDecoder(strings.NewReader(*tokenResponseString))
	if err := jsonParser.Decode(&tokenResponse); err != nil {
		log.Printf("Failed to parse token response: %v", err)
		return TokenResponse{}, err
	}
	tokenResponse.Timestamp = time.Now().Unix()
	return tokenResponse, nil
}

func LoadTokenResponseString(profileName string) (*string, error) {
	secret, err := keyring.Get(KEYRING_SERVICE_NAME, profileName)
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func SaveTokenResponseString(profileName string, tokenResponseString *string) {
	err := keyring.Set(KEYRING_SERVICE_NAME, profileName, *tokenResponseString)
	if err != nil {
		log.Printf("Failed to save token response to keyring: %v", err.Error())
	}
}
