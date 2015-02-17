package main

import (
	"github.com/grategames/ini"
)

const DEFAULT_STORAGE = "~/.aurl/tokens"

const (
	ACCESS_TOKEN  = "access_token"
	TOKEN_TYPE    = "token_type"
	REFRESH_TOKEN = "refresh_token"
	EXPIRY        = "expiry"
)

func init() {
	config.DefaultSection = ""
}

func LoadValue(section string, key string) (string, error) {
	c, err := config.ReadConfigFile(expandPath(DEFAULT_STORAGE))
	if err != nil {
		return "", err
	}
	return c.GetString(section, key)
}

func LoadValues(section string) (map[string]string, error) {
	c, err := config.ReadConfigFile(expandPath(DEFAULT_STORAGE))
	if err != nil {
		return nil, err
	}
	keys, err := c.GetOptions(section)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(keys))
	Tracef("keys %v", keys)
	for _, key := range keys {
		if value, err := c.GetString(section, key); err != nil {
			Tracef("get error: %v", err)
		} else {
			Tracef("%s = %s", key, value)
			result[key] = value
		}
	}

	return result, nil
}

func SaveValue(section string, key string, value string) bool {
	values := make(map[string]string)
	values[key] = value
	return SaveValues(section, values)
}

func SaveValues(section string, values map[string]string) bool {
	filename := expandPath(DEFAULT_STORAGE)

	c, err := config.ReadConfigFile(filename)
	if err != nil {
		c = config.NewConfigFile()
	}

	for key, value := range values {
		c.AddOption(section, key, value)
	}
	err = c.WriteConfigFile(filename, 0600, "")
	return err == nil
}
