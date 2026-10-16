package cli

import (
	"errors"
	"fmt"
	"log"

	"github.com/alecthomas/kingpin/v2"
	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/util"
	"github.com/classmethod/aurl/vault"
)

type AddCommandInput struct {
	ProfileName string
	force       bool
}

func ConfigureAddCommand(app *kingpin.Application, a *Aurl) {
	input := AddCommandInput{}

	cmd := app.Command("add", "Add credentials to the secure keystore.")

	cmd.Arg("profile", "Name of the profile to add (prompted when omitted).").
		StringVar(&input.ProfileName)

	cmd.Flag("force", "Force adding even if the profile already exists in the config file.").
		BoolVar(&input.force)

	cmd.Action(func(c *kingpin.ParseContext) error {
		if input.ProfileName == "" {
			var err error
			if input.ProfileName, err = util.TerminalPrompt("Enter Profile Name: "); err != nil {
				return err
			}
			if input.ProfileName == "" {
				return errors.New("profile name is required")
			}
		}
		aurlConfigFile, err := a.AurlConfigFile()
		if err != nil {
			return err
		}
		keyring, err := a.Keyring()
		if err != nil {
			return err
		}

		kingpin.FatalIfError(AddCommand(input, keyring, aurlConfigFile), "add")
		return nil
	})
}

func AddCommand(input AddCommandInput, keyring keyring.Keyring, aurlConfigFile *vault.ConfigFile) error {
	if _, hasProfile := aurlConfigFile.ProfileSection(input.ProfileName); hasProfile && !input.force {
		return fmt.Errorf("Profile %q already exists in config at %s (use --force to override)", input.ProfileName, aurlConfigFile.Path)
	}

	var grantType, authzServerAuthEndpoint, authzServerTokenEndpoint, redirectURI, clientId, clientSecret, username, password, scope, contentType, userAgent, baseURL string
	var err error

	if grantType, err = selectFromList("Select Grant Type:", []string{"authorization_code", "implicit", "password", "client_credentials"}); err != nil {
		return err
	}

	// Common fields for all grant types
	if clientId, err = util.TerminalSecretPrompt("Enter Client ID: "); err != nil {
		return err
	}
	if clientSecret, err = util.TerminalSecretPrompt("Enter Client Secret: "); err != nil {
		return err
	}

	// Grant type specific fields
	switch grantType {
	case "authorization_code":
		if authzServerAuthEndpoint, err = util.TerminalPrompt("Enter Authz Server Auth Endpoint: "); err != nil {
			return err
		}
		if authzServerTokenEndpoint, err = util.TerminalPrompt("Enter Authz Server Token Endpoint: "); err != nil {
			return err
		}
		if redirectURI, err = util.TerminalPrompt("Enter Redirect URI: "); err != nil {
			return err
		}
		if scope, err = util.TerminalPrompt("Enter Scopes (space separated): "); err != nil {
			return err
		}

	case "implicit":
		if authzServerAuthEndpoint, err = util.TerminalPrompt("Enter Authz Server Auth Endpoint: "); err != nil {
			return err
		}
		if redirectURI, err = util.TerminalPrompt("Enter Redirect URI: "); err != nil {
			return err
		}
		if scope, err = util.TerminalPrompt("Enter Scopes (space separated): "); err != nil {
			return err
		}

	case "password":
		if authzServerTokenEndpoint, err = util.TerminalPrompt("Enter Authz Server Token Endpoint: "); err != nil {
			return err
		}
		if username, err = util.TerminalPrompt("Enter Username: "); err != nil {
			return err
		}
		if password, err = util.TerminalSecretPrompt("Enter Password: "); err != nil {
			return err
		}
		if scope, err = util.TerminalPrompt("Enter Scopes (space separated): "); err != nil {
			return err
		}

	case "client_credentials":
		if authzServerTokenEndpoint, err = util.TerminalPrompt("Enter Authz Server Token Endpoint: "); err != nil {
			return err
		}
		if scope, err = util.TerminalPrompt("Enter Scopes (space separated): "); err != nil {
			return err
		}

	default:
		return fmt.Errorf("Unknown grant type: %s", grantType)
	}

	// Common optional fields
	if contentType, err = util.TerminalPromptWithDefault("Enter Content Type (default: application/json): ", "application/json"); err != nil {
		return err
	}
	if userAgent, err = util.TerminalPromptWithDefault("Enter User Agent (default: aurl): ", "aurl"); err != nil {
		return err
	}
	if baseURL, err = util.TerminalPrompt("Enter Base URL (optional, e.g. https://api.example.com): "); err != nil {
		return err
	}

	creds := vault.Credentials{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Username:     username,
		Password:     password,
	}
	newProfileSection := vault.ProfileSection{
		Name:                    input.ProfileName,
		GrantType:               grantType,
		AuthServerAuthEndpoint:  authzServerAuthEndpoint,
		AuthServerTokenEndpoint: authzServerTokenEndpoint,
		Redirect:                redirectURI,
		Scope:                   scope,
		ContentType:             contentType,
		UserAgent:               userAgent,
		BaseURL:                 baseURL,
	}
	if err := saveProfile(keyring, aurlConfigFile, newProfileSection, creds); err != nil {
		return err
	}
	fmt.Printf("Added credentials to profile %q in vault\n", input.ProfileName)
	return nil
}

func saveProfile(keyring keyring.Keyring, aurlConfigFile *vault.ConfigFile, section vault.ProfileSection, creds vault.Credentials) error {
	ckr := &vault.CredentialKeyring{Keyring: keyring}
	if err := ckr.Set(section.Name, creds); err != nil {
		return fmt.Errorf("Error storing credentials in keyring: %w", err)
	}

	log.Printf("Adding profile %s to config at %s", section.Name, aurlConfigFile.Path)
	if err := aurlConfigFile.Add(section); err != nil {
		return fmt.Errorf("Error adding profile: %w", err)
	}

	// Remove any existing tokens for the profile
	tkr := &vault.TokenKeyring{Keyring: keyring}
	if err := tkr.Remove(section.Name); err != nil {
		fmt.Printf("Warning: Failed to remove existing token for profile %q: %v\n", section.Name, err)
	}

	return nil
}
