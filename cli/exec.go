package cli

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/request"
	"github.com/classmethod/aurl/util"
	"github.com/classmethod/aurl/vault"
)

type ExecCommandInput struct {

	// for aurl options
	ProfileName string
	RenewToken  bool

	// for curl options
	Method       string
	Headers      []string
	Data         string
	Insecure     bool
	PrintBody    bool
	PrintHeaders bool

	TargetUrl string
}

func ConfigureExecCommand(app *kingpin.Application, a *Aurl) {
	input := ExecCommandInput{}

	cmd := app.Command("curl", "Send an HTTP request with the profile's token (default command: aurl <profile> <url>). Prompts for profile and url when omitted.").Default()

	cmd.Arg("profile", "Name of the profile to use.").
		HintAction(a.MustGetProfileNames).
		StringVar(&input.ProfileName)
	cmd.Flag("renew-token", "Force renewal of access token even if a valid token exists in the keyring.").
		BoolVar(&input.RenewToken)

	cmd.Flag("request", "Set HTTP request method. (default: \"GET\")").
		Short('X').
		Default("GET").
		StringVar(&input.Method)
	cmd.Flag("header", "Add HTTP headers to the request.").
		Short('H').
		PlaceHolder("HEADER:VALUE").
		StringsVar(&input.Headers)
	cmd.Flag("data", "Set HTTP request body.").
		Short('d').
		StringVar(&input.Data)
	cmd.Flag("insecure", "Disable SSL certificate verification.").
		Short('k').
		BoolVar(&input.Insecure)
	cmd.Flag("print-body", "Enable printing response body to stdout. (default: enabled, try --no-print-body)").
		Default("true").
		BoolVar(&input.PrintBody)
	cmd.Flag("print-headers", "Enable printing response headers JSON to stdout. (default: disabled, try --no-print-headers)").
		BoolVar(&input.PrintHeaders)

	cmd.Arg("url", "The URL to request").
		StringVar(&input.TargetUrl)

	cmd.Action(func(c *kingpin.ParseContext) (err error) {
		aurlConfigFile, err := a.AurlConfigFile()
		if err != nil {
			return err
		}

		if input.ProfileName == "" {
			if input.ProfileName, err = selectProfile(aurlConfigFile.ProfileNames(), aurlConfigFile.Path); err != nil {
				return err
			}
		}
		section, ok := aurlConfigFile.ProfileSection(input.ProfileName)
		if !ok {
			showUsage, err := profileNotFound(input.ProfileName, aurlConfigFile.Path)
			if showUsage {
				app.Usage(nil)
			}
			return err
		}
		if input.TargetUrl == "" {
			if input.TargetUrl, err = util.TerminalPrompt(urlPromptMessage(section.BaseURL)); err != nil {
				return err
			}
			if input.TargetUrl == "" {
				return errors.New("URL is required")
			}
		}

		// Opened after the prompts so op backends do not ask for a token first
		keyring, err := a.Keyring()
		if err != nil {
			return err
		}

		kingpin.FatalIfError(ExecCommand(input, keyring, aurlConfigFile), "curl")
		return nil
	})
}

func selectProfile(names []string, configPath string) (string, error) {
	if len(names) == 0 {
		if hasV1Profiles() {
			return "", fmt.Errorf("Config file %s has no profiles, but the aurl v1 profiles file ~/.aurl/profiles has some. Run `aurl migrate` to import them.", configPath)
		}
		return "", fmt.Errorf("Config file %s has no profiles. Run `aurl add` to create one.", configPath)
	}
	return selectFromList("Select profile:", names)
}

// profileNotFound explains why name is not a profile; showUsage is set when name is likely a mistyped command.
func profileNotFound(name, configPath string) (showUsage bool, err error) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		return false, fmt.Errorf("%q is a URL, not a profile (%w)", name, ErrV1Style)
	case inV1Profiles(name):
		return false, fmt.Errorf("Profile %q not found in %s, but it was found in ~/.aurl/profiles (aurl v1). Run `aurl migrate` to import it.", name, configPath)
	}
	return true, fmt.Errorf("unknown profile or command %q", name)
}

func urlPromptMessage(baseURL string) string {
	if baseURL == "" {
		return "Enter URL (no base_url set): "
	}
	return fmt.Sprintf("Enter URL or path (base_url: %s): ", baseURL)
}

func parseProfileChoice(names []string, answer string) (string, error) {
	n, err := strconv.Atoi(answer)
	if err != nil || n < 1 || n > len(names) {
		return "", fmt.Errorf("invalid number %q: choose 1-%d", answer, len(names))
	}
	return names[n-1], nil
}

func ExecCommand(input ExecCommandInput, keyring keyring.Keyring, aurlConfigFile *vault.ConfigFile) (err error) {

	config, err := vault.NewConfigLoader(aurlConfigFile, input.ProfileName).GetProfileConfig(input.ProfileName)
	if err != nil {
		if inV1Profiles(input.ProfileName) {
			return fmt.Errorf("Error loading config: %w\nProfile %q was found in ~/.aurl/profiles (aurl v1). Run `aurl migrate` to import it.", err, input.ProfileName)
		}
		return fmt.Errorf("Error loading config: %w", err)
	}
	if input.TargetUrl, err = resolveURL(config.BaseURL, input.TargetUrl); err != nil {
		return err
	}

	var tokenInfo *vault.TokenInfo
	// Load previous token from keyring unless RenewToken is specified
	if !input.RenewToken {
		tkr := &vault.TokenKeyring{Keyring: keyring}
		if tokenInfo, err = tkr.Get(input.ProfileName); err != nil {
			log.Printf("Previous token not found in keyring: %v", err.Error())
		}
	}

	execution := &request.Request{
		Name: input.ProfileName,

		Config:       config,
		TokenInfo:    tokenInfo,
		Method:       &input.Method,
		Headers:      &input.Headers,
		Data:         &input.Data,
		Insecure:     &input.Insecure,
		PrintBody:    &input.PrintBody,
		PrintHeaders: &input.PrintHeaders,

		TargetUrl: &input.TargetUrl,
	}

	kingpin.FatalIfError(execution.Execute(keyring), "Request failed")

	return nil
}

func resolveURL(baseURL, target string) (string, error) {
	lower := strings.ToLower(target)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return target, nil
	}
	if baseURL == "" {
		return "", fmt.Errorf("relative URL %q requires base_url in the profile", target)
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(target, "/"), nil
}
