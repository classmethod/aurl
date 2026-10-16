package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/byteness/keyring"
	"github.com/classmethod/aurl/vault"
	goini "github.com/rakyll/goini"
)

type MigrateCommandInput struct {
	ProfilesFile string
	Force        bool
}

func ConfigureMigrateCommand(app *kingpin.Application, a *Aurl) {
	input := MigrateCommandInput{}

	cmd := app.Command("migrate", "Import aurl v1 profiles (~/.aurl/profiles) into the config file and the secure keystore.")

	cmd.Flag("force", "Overwrite configuration that already exist in the config file.").
		BoolVar(&input.Force)
	cmd.Flag("profiles-file", "Path to the aurl v1 profiles file. (default: ~/.aurl/profiles)").
		PlaceHolder("PATH").
		StringVar(&input.ProfilesFile)

	cmd.Action(func(c *kingpin.ParseContext) error {
		keyring, err := a.Keyring()
		if err != nil {
			return err
		}
		aurlConfigFile, err := a.AurlConfigFile()
		if err != nil {
			return err
		}

		kingpin.FatalIfError(MigrateCommand(input, keyring, aurlConfigFile), "migrate")
		return nil
	})
}

func MigrateCommand(input MigrateCommandInput, keyring keyring.Keyring, aurlConfigFile *vault.ConfigFile) error {
	path := input.ProfilesFile
	if path == "" {
		var err error
		if path, err = v1ProfilesPath(); err != nil {
			return err
		}
	}

	v1, err := goini.Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("aurl v1 profiles file not found at %s", path)
	}
	if err != nil {
		return fmt.Errorf("Error reading %s: %w", path, err)
	}

	names := make([]string, 0, len(v1))
	for name := range v1 {
		// Keys before the first section land in "", which v1 could not use as a profile.
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if _, exists := aurlConfigFile.ProfileSection(name); exists && !input.Force {
			fmt.Printf("%s: skipped (already exists, use --force)\n", name)
			continue
		}
		section, creds, unknown := convertV1Profile(name, v1[name])
		for _, k := range unknown {
			fmt.Fprintf(os.Stderr, "Warning: profile %q: ignoring unknown key %q\n", name, k)
		}
		if err := saveProfile(keyring, aurlConfigFile, section, creds); err != nil {
			return fmt.Errorf("Error migrating profile %q: %w", name, err)
		}
		fmt.Printf("%s: migrated\n", name)
	}
	fmt.Printf("\n%s was left untouched. It contains plain-text secrets, so delete it once you have verified the migration.\n", path)
	return nil
}

func convertV1Profile(name string, keys map[string]string) (vault.ProfileSection, vault.Credentials, []string) {
	known := map[string]bool{}
	get := func(key, def string) string {
		known[key] = true
		// v1 applies defaults only when the key is absent, not when it is empty
		if v, ok := keys[key]; ok {
			return v
		}
		return def
	}

	section := vault.ProfileSection{
		Name:                    name,
		GrantType:               get("grant_type", "authorization_code"),
		AuthServerAuthEndpoint:  get("auth_server_auth_endpoint", ""),
		AuthServerTokenEndpoint: get("auth_server_token_endpoint", ""),
		Redirect:                get("redirect", ""),
		Scope:                   get("scopes", "root"),
		ContentType:             get("default_content_type", ""),
		UserAgent:               get("default_user_agent", ""),
	}
	creds := vault.Credentials{
		ClientId:     get("client_id", "aurl"),
		ClientSecret: get("client_secret", "aurl"),
		Username:     get("username", ""),
		Password:     get("password", ""),
	}

	var unknown []string
	for k := range keys {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	return section, creds, unknown
}

// inV1Profiles matches name exactly like v1 did; goini has already lowercased the section names.
func inV1Profiles(name string) bool {
	path, err := v1ProfilesPath()
	if err != nil || name == "" {
		return false
	}
	v1, err := goini.Load(path)
	if err != nil {
		return false
	}
	_, ok := v1[name]
	return ok
}

func hasV1Profiles() bool {
	path, err := v1ProfilesPath()
	if err != nil {
		return false
	}
	v1, err := goini.Load(path)
	if err != nil {
		return false
	}
	for name := range v1 {
		if name != "" {
			return true
		}
	}
	return false
}

func v1ProfilesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aurl", "profiles"), nil
}

// IsV1Style reports whether args look like an aurl v1 invocation (aurl [-p profile] <url>).
func IsV1Style(args []string) bool {
	hasSubcommand, hasURL := false, false
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-p") || arg == "--profile" || strings.HasPrefix(arg, "--profile="):
			return true
		case arg == "exec" || arg == "add" || arg == "migrate" || arg == "help":
			hasSubcommand = true
		case strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://"):
			hasURL = true
		}
	}
	return !hasSubcommand && hasURL
}
