package cli

import (
	"fmt"
	"log"

	"github.com/alecthomas/kingpin/v2"
	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/request"
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

	cmd := app.Command("exec", "Execute a command with Token.")

	cmd.Arg("profile", "Name of the profile to use.").
		Required().
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
		Required().
		StringVar(&input.TargetUrl)

	cmd.Action(func(c *kingpin.ParseContext) (err error) {
		keyring, err := a.Keyring()
		if err != nil {
			return err
		}
		aurlConfigFile, err := a.AurlConfigFile()
		if err != nil {
			return err
		}

		kingpin.FatalIfError(ExecCommand(input, keyring, aurlConfigFile), "exec")
		return nil
	})
}

func ExecCommand(input ExecCommandInput, keyring keyring.Keyring, aurlConfigFile *vault.ConfigFile) (err error) {

	config, err := vault.NewConfigLoader(aurlConfigFile, input.ProfileName).GetProfileConfig(input.ProfileName)
	if err != nil {
		return fmt.Errorf("Error loading config: %w", err)
	}

	ckr := &vault.CredentialKeyring{Keyring: keyring}
	creds, credsErr := ckr.Get(input.ProfileName)
	if credsErr != nil {
		return fmt.Errorf("Failed to get credentials: %w", credsErr)
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
		Credentials:  creds,
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
