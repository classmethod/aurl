package vault

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	ini "gopkg.in/ini.v1"
)

const (
	defaultSectionName = "default"
)

func init() {
	ini.PrettyFormat = false
}

type Config struct {
	Name                  string
	GrantType             string
	AuthorizationEndpoint string
	TokenEndpoint         string
	RedirectURI           string
	Scope                 string
	ContentType           string
	UserAgent             string
}

type ConfigFile struct {
	Path    string
	iniFile *ini.File
}

func configPath() (string, error) {
	file := os.Getenv("AURL_CONFIG_FILE")
	if file == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		file = filepath.Join(home, "/.aurl/config")
	} else {
		log.Printf("Using AURL_CONFIG_FILE value: %s", file)
	}
	return file, nil
}

func LoadConfig() (*ConfigFile, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	config := &ConfigFile{
		Path: path,
	}
	if _, err := os.Stat(path); err == nil {
		if parseErr := config.parseFile(); parseErr != nil {
			return nil, parseErr
		}
	} else {
		log.Printf("Config file %s doesn't exist so lets create it", path)
		err := createConfigFilesIfMissing(path)
		if err != nil {
			return nil, err
		}
		if parseErr := config.parseFile(); parseErr != nil {
			return nil, parseErr
		}
	}
	return config, nil
}

func createConfigFilesIfMissing(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0700)
		if err != nil {
			return err
		}
		log.Printf("Config directory %s created", dir)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		newFile, err := os.Create(path)
		if err != nil {
			log.Printf("Config file %s not created", path)
			return err
		}
		newFile.Close()
		log.Printf("Config file %s created", path)
	}
	return nil
}

func (c *ConfigFile) parseFile() error {
	log.Printf("Parsing config file %s", c.Path)

	f, err := ini.LoadSources(ini.LoadOptions{
		AllowNestedValues:   true,
		InsensitiveSections: false,
		InsensitiveKeys:     true,
	}, c.Path)
	if err != nil {
		return fmt.Errorf("Error parsing config file %s: %w", c.Path, err)
	}
	c.iniFile = f
	return nil
}

// ProfileSection is a profile section of the config file
type ProfileSection struct {
	Name                    string `ini:"-"`
	GrantType               string `ini:"grant_type"`
	AuthServerAuthEndpoint  string `ini:"auth_server_auth_endpoint"`
	AuthServerTokenEndpoint string `ini:"auth_server_token_endpoint"`
	Redirect                string `ini:"redirect"`
	Scope                   string `ini:"scopes"`
	ContentType             string `ini:"content_type"`
	UserAgent               string `ini:"user_agent"`
}

func (s ProfileSection) IsEmpty() bool {
	s.Name = ""
	return s == ProfileSection{}
}

// ProfileSections returns all the profile sections in the config
func (c *ConfigFile) ProfileSections() []ProfileSection {
	result := []ProfileSection{}

	if c.iniFile == nil {
		return result
	}
	for _, section := range c.iniFile.SectionStrings() {
		if section == defaultSectionName {
			profile, _ := c.ProfileSection(section)

			// ignore the default profile if it's empty
			if section == defaultSectionName && profile.IsEmpty() {
				continue
			}

			result = append(result, profile)
		} else {
			log.Printf("Unrecognised ini file section: %s", section)
			continue
		}
	}

	return result
}

// ProfileSection returns the profile section with the matching name. If there isn't any,
// an empty profile with the provided name is returned, along with false.
func (c *ConfigFile) ProfileSection(name string) (ProfileSection, bool) {
	profile := ProfileSection{
		Name: name,
	}
	if c.iniFile == nil {
		return profile, false
	}
	// default profile name has a slightly different section format
	sectionName := name
	if name == defaultSectionName {
		sectionName = defaultSectionName
	}
	section, err := c.iniFile.GetSection(sectionName)
	if err != nil {
		return profile, false
	}
	if err = section.MapTo(&profile); err != nil {
		panic(err)
	}
	return profile, true
}

func (c *ConfigFile) Save() error {
	return c.iniFile.SaveTo(c.Path)
}

// Add the profile to the configuration file
func (c *ConfigFile) Add(profile ProfileSection) error {
	if c.iniFile == nil {
		return errors.New("No iniFile to add to")
	}
	// default profile name has a slightly different section format
	section, err := c.iniFile.NewSection(profile.Name)
	if err != nil {
		return fmt.Errorf("Error creating section %q: %v", profile.Name, err)
	}
	if err = section.ReflectFrom(&profile); err != nil {
		return fmt.Errorf("Error mapping profile to ini file: %v", err)
	}
	return c.Save()
}

func (c *ConfigFile) ProfileNames() []string {
	profileNames := []string{}
	for _, profile := range c.ProfileSections() {
		profileNames = append(profileNames, profile.Name)
	}
	return profileNames
}

type ConfigLoader struct {
	File          *ConfigFile
	ActiveProfile string
}

func NewConfigLoader(file *ConfigFile, activeProfile string) *ConfigLoader {
	return &ConfigLoader{
		File:          file,
		ActiveProfile: activeProfile,
	}
}

func (cl *ConfigLoader) GetProfileConfig(profileName string) (*Config, error) {

	profileSection, ok := cl.File.ProfileSection(profileName)
	if !ok {
		return nil, fmt.Errorf("Profile '%s' not found in config file", profileName)
	}
	contentType := profileSection.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	userAgent := profileSection.UserAgent
	if userAgent == "" {
		userAgent = "aurl"
	}

	config := Config{
		Name:                  profileName,
		GrantType:             profileSection.GrantType,
		AuthorizationEndpoint: profileSection.AuthServerAuthEndpoint,
		TokenEndpoint:         profileSection.AuthServerTokenEndpoint,
		RedirectURI:           profileSection.Redirect,
		Scope:                 profileSection.Scope,
		ContentType:           contentType,
		UserAgent:             userAgent,
	}

	return &config, nil
}
