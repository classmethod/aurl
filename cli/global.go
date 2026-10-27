package cli

import (
	"fmt"
	"io"
	"log"
	"slices"

	"github.com/alecthomas/kingpin/v2"
	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/util"
	"github.com/classmethod/aurl/vault"
)

type Aurl struct {
	Verbose        bool
	aurlConfigFile *vault.ConfigFile
	KeyringConfig  keyring.Config
	keyringImpl    keyring.Keyring
	KeyringBackend string
}

// OS-native keyrings and 1Password only; the rest (file, pass, kwallet, ...) are unconfigured in aurl
var supportedBackends = []keyring.BackendType{
	keyring.KeychainBackend,
	keyring.WinCredBackend,
	keyring.SecretServiceBackend,
	keyring.OPDesktopBackend,
	keyring.OPBackend,
	keyring.OPConnectBackend,
}

var keyringConfigDefaults = keyring.Config{
	ServiceName:              "aurl",
	LibSecretCollectionName:  "aurl",
	KWalletAppID:             "aurl",
	KWalletFolder:            "aurl",
	WinCredPrefix:            "aurl",
	KeychainTrustApplication: true,
	OPItemTitlePrefix:        "aurl",
	OPItemTag:                "aurl",
	OPTokenEnv:               "AURL_OP_SERVICE_ACCOUNT_TOKEN",
	OPConnectTokenEnv:        "AURL_OP_CONNECT_TOKEN",
	OPTokenFunc:              util.TerminalSecretPrompt,
}

// Current and planned subcommand names; a profile with one of them could not be reached as `aurl <profile>`
var reservedNames = []string{"add", "migrate", "help", "curl", "exec", "run", "list", "remove", "clear", "login", "export", "rotate", "proxy", "token"}

func isReserved(name string) bool {
	return slices.Contains(reservedNames, name)
}

func (a *Aurl) Keyring() (keyring.Keyring, error) {
	if a.keyringImpl == nil {
		if a.KeyringBackend != "" {
			a.KeyringConfig.AllowedBackends = []keyring.BackendType{keyring.BackendType(a.KeyringBackend)}
		}
		var err error
		a.keyringImpl, err = keyring.Open(a.KeyringConfig)
		if err != nil {
			return nil, err
		}
	}

	return a.keyringImpl, nil
}

func (a *Aurl) AurlConfigFile() (*vault.ConfigFile, error) {
	if a.aurlConfigFile == nil {
		var err error
		a.aurlConfigFile, err = vault.LoadConfig()
		if err != nil {
			return nil, err
		}
	}

	return a.aurlConfigFile, nil
}

func (a *Aurl) MustGetProfileNames() []string {
	config, err := a.AurlConfigFile()
	if err != nil {
		log.Fatalf("Error loading aurl config: %s", err.Error())
	}
	return config.ProfileNames()
}

func ConfigureGlobals(app *kingpin.Application) *Aurl {
	a := &Aurl{
		KeyringConfig: keyringConfigDefaults,
	}

	backendsAvailable := []string{}
	for _, backendType := range keyring.AvailableBackends() {
		if slices.Contains(supportedBackends, backendType) {
			backendsAvailable = append(backendsAvailable, string(backendType))
		}
	}

	app.Flag("verbose", "Enable verbose logging to stderr.").
		Short('v').
		BoolVar(&a.Verbose)

	app.Flag("backend", fmt.Sprintf("Secret backend to use %v", backendsAvailable)).
		Default(backendsAvailable[0]).
		Envar("AURL_BACKEND").
		EnumVar(&a.KeyringBackend, backendsAvailable...)

	app.Flag("op-timeout", "Timeout for 1Password API operations (op / op-desktop only)").
		Default("15s").
		Envar("AURL_OP_TIMEOUT").
		DurationVar(&a.KeyringConfig.OPTimeout)

	app.Flag("op-vault-id", "UUID of the 1Password vault").
		Envar("AURL_OP_VAULT_ID").
		StringVar(&a.KeyringConfig.OPVaultID)

	app.Flag("op-desktop-account-id", "1Password account name or UUID for the desktop app integration").
		Envar("AURL_OP_DESKTOP_ACCOUNT_ID").
		StringVar(&a.KeyringConfig.OPDesktopAccountID)

	app.Flag("op-connect-host", "1Password Connect server HTTP(S) URI").
		Envar("AURL_OP_CONNECT_HOST").
		StringVar(&a.KeyringConfig.OPConnectHost)

	app.PreAction(func(c *kingpin.ParseContext) error {
		keyring.Debug = a.Verbose
		if a.Verbose {
			log.SetOutput(log.Writer())
			log.SetPrefix("**** ")
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		} else {
			log.SetOutput(io.Discard)
		}

		log.Printf("%s %s", app.Model().Name, app.Model().Version)
		return nil
	})

	return a
}
