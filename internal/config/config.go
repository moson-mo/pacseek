package config

import (
	"encoding/json"
	"os"
	"path"
)

// Settings is a structure containing our configuration data
type Settings struct {
	AurRpcUrl               string
	AurTimeout              int
	AurSearchDelay          int
	AurUseDifferentCommands bool
	AurInstallCommand       string
	AurUpgradeCommand       string
	DisableAur              bool
	MaxResults              int
	PacmanDbPath            string
	PacmanConfigPath        string
	InstallCommand          string
	UninstallCommand        string
	SysUpgradeCommand       string
	SearchMode              string
	SearchBy                string
	CacheExpiry             int
	DisableCache            bool
	ColorScheme             string
	BorderStyle             string
	ShowPkgbuildCommand     string
	ShowPkgbuildInternally  bool
	ComputeRequiredBy       bool
	GlyphStyle              string
	DisableNewsFeed         bool
	FeedURLs                string
	FeedMaxItems            int
	SaveTilingProportion    bool
	LeftProportion          int
	colors                  Colors
	glyphs                  Glyphs
}

// Defaults returns the default settings
func Defaults() *Settings {
	s := Settings{
		AurRpcUrl:              "https://server.moson.rocks/rpc",
		AurTimeout:             5000,
		AurSearchDelay:         500,
		DisableAur:             false,
		MaxResults:             500,
		PacmanDbPath:           "/var/lib/pacman/",
		PacmanConfigPath:       "/etc/pacman.conf",
		InstallCommand:         "yay -S",
		UninstallCommand:       "yay -Rs",
		SearchMode:             "Contains",
		SysUpgradeCommand:      "yay",
		SearchBy:               "Name",
		CacheExpiry:            10,
		DisableCache:           false,
		ColorScheme:            defaultColorScheme,
		BorderStyle:            "Double",
		colors:                 colorSchemes[defaultColorScheme],
		ShowPkgbuildCommand:    "curl -s \"{url}\"|less",
		ShowPkgbuildInternally: true,
		ComputeRequiredBy:      false,
		GlyphStyle:             defaultGlyphStyle,
		DisableNewsFeed:        false,
		FeedURLs:               "https://archlinux.org/feeds/news/",
		FeedMaxItems:           5,
		SaveTilingProportion:   false,
		LeftProportion:         4,
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

	if err = os.WriteFile(path.Join(confPath, "config.json"), b, 0644); err != nil {
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

	b, err := os.ReadFile(confFile)
	if err != nil {
		return Defaults(), err
	}
	ret := Settings{}
	if err = json.Unmarshal(b, &ret); err != nil {
		return Defaults(), err
	}
	ret.applyUpgradeFixes()
	ret.SetColorScheme(ret.ColorScheme)
	ret.SetBorderStyle(ret.BorderStyle)
	ret.SetGlyphStyle(ret.GlyphStyle)
	return &ret, nil
}

// fix settings in case of version upgrades (e.g. new config options that have to be set)
func (s *Settings) applyUpgradeFixes() {
	fixApplied := false
	def := Defaults()

	// search mode: added with 0.1.2
	if s.SearchMode == "" {
		s.SearchMode = def.SearchMode
		fixApplied = true
	}

	// sysupgrade command: added with 0.2.4
	if s.SysUpgradeCommand == "" {
		s.SysUpgradeCommand = def.SysUpgradeCommand
		fixApplied = true
	}

	// search by: added with 0.2.7
	if s.SearchBy == "" {
		s.SearchBy = def.SearchBy
		fixApplied = true
	}

	// cache expiry: added with 1.1.0
	if s.CacheExpiry == 0 {
		s.CacheExpiry = def.CacheExpiry
		fixApplied = true
	}

	// color scheme: added with 1.4.2
	if s.ColorScheme == "" {
		s.ColorScheme = def.ColorScheme
		fixApplied = true
	}

	// border style: added with 1.4.2
	if s.BorderStyle == "" {
		s.BorderStyle = def.BorderStyle
		fixApplied = true
	}

	// show PKGBUILD command: added with 1.4.7
	if s.ShowPkgbuildCommand == "" {
		s.ShowPkgbuildCommand = def.ShowPkgbuildCommand
		fixApplied = true
	}

	// Glyph style added with 1.6.9
	if s.GlyphStyle == "" {
		s.GlyphStyle = def.GlyphStyle
		fixApplied = true
	}

	// News feed added with 1.7.0
	if s.FeedURLs == "" {
		s.FeedURLs = def.FeedURLs
		fixApplied = true
	}
	if s.FeedMaxItems == 0 {
		s.FeedMaxItems = def.FeedMaxItems
		fixApplied = true
	}

	// save config file when we applied changes
	if fixApplied {
		s.Save()
	}
}
