package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

// Settings is a structure containing our configuration data
type Settings struct {
	AurRpcUrl        string
	AurTimeout       int
	AurSearchDelay   int
	MaxResults       int
	PacmanDbPath     string
	PacmanConfigPath string
	InstallCommand   string
	UninstallCommand string
	SearchMode       string
}

// Defaults returns the default settings
func Defaults() *Settings {
	s := Settings{
		AurRpcUrl:        "https://server.moson.rocks/rpc",
		AurTimeout:       5000,
		AurSearchDelay:   500,
		MaxResults:       100,
		PacmanDbPath:     "/var/lib/pacman/",
		PacmanConfigPath: "/etc/pacman.conf",
		InstallCommand:   "yay -S",
		UninstallCommand: "yay -Rs",
		SearchMode:       "StartsWith",
	}

	return &s
}

// Save is creating / overwriting our configuration file ./config/rpcsearch/config.json
func (s *Settings) Save() error {
	b, err := json.MarshalIndent(s, "", "	")
	if err != nil {
		return err
	}

	confPath, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	confPath = path.Join(confPath, "/pacseek")
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		os.MkdirAll(confPath, 0755)
	} else if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path.Join(confPath, "config.json"), b, 0644); err != nil {
		return err
	}
	return nil
}

// Load is loading our settings from the config file
func Load() (*Settings, error) {
	confFile, err := os.UserConfigDir()
	if err != nil {
		return Defaults(), err
	}
	confFile = path.Join(confFile, "/pacseek/config.json")

	b, err := ioutil.ReadFile(confFile)
	if err != nil {
		return Defaults(), err
	}
	ret := Settings{}
	if err = json.Unmarshal(b, &ret); err != nil {
		return Defaults(), err
	}
	return &ret, nil
}
