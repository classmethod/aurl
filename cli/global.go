package cli

import (
	"fmt"
	"io"
	"log"

	"github.com/alecthomas/kingpin/v2"
	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/vault"
)

type Aurl struct {
	Verbose        bool
	aurlConfigFile *vault.ConfigFile
	KeyringConfig  keyring.Config
	keyringImpl    keyring.Keyring
	KeyringBackend string
}

var keyringConfigDefaults = keyring.Config{
	ServiceName:              "aurl",
	LibSecretCollectionName:  "aurl",
	KWalletAppID:             "aurl",
	KWalletFolder:            "aurl",
	WinCredPrefix:            "aurl",
	KeychainTrustApplication: true,
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
		backendsAvailable = append(backendsAvailable, string(backendType))
	}

	app.Flag("verbose", "Enable verbose logging to stderr.").
		Short('v').
		BoolVar(&a.Verbose)

	app.Flag("backend", fmt.Sprintf("Secret backend to use %v", backendsAvailable)).
		Default(backendsAvailable[0]).
		EnumVar(&a.KeyringBackend, backendsAvailable...)

	app.PreAction(func(c *kingpin.ParseContext) error {
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
