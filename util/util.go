package util

import "github.com/byteness/aws-vault/v7/prompt"

// promptWithDefault prompts the user with a message and returns the input or default value if empty
func TerminalPromptWithDefault(message, defaultValue string) (string, error) {
	value, err := prompt.TerminalPrompt(message)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func TerminalPrompt(message string) (string, error) {
	return prompt.TerminalPrompt(message)
}

func TerminalSecretPrompt(message string) (string, error) {
	return prompt.TerminalSecretPrompt(message)
}
