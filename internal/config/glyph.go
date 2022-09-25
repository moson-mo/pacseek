package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
)

type Glyphs struct {
	Package      string
	Installed    string
	NotInstalled string
	PrefixState  string
	SuffixState  string
	Settings     string
	Pkgbuild     string
	Help         string
	Upgrades     string
}

// default glyph style
const (
	defaultGlyphStyle = "Angled-No-X"
)

var (
	glyphStyles = map[string]Glyphs{
		"Plain": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Angled": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "[",
			SuffixState:  "]",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Round": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "(",
			SuffixState:  ")",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Curly": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "{",
			SuffixState:  "}",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Pipes": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "|",
			SuffixState:  "|",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"ASCII": {
			Package:      "",
			Installed:    "Y",
			NotInstalled: "-",
		},
		"Plain-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Angled-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "[",
			SuffixState:  "]",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Round-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "(",
			SuffixState:  ")",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Curly-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "{",
			SuffixState:  "}",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"Pipes-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "|",
			SuffixState:  "|",
			Settings:     "📝 ",
			Pkgbuild:     "📄",
			Help:         "📖 ",
			Upgrades:     "🔁 ",
		},
		"ASCII-No-X": {
			Package:      "",
			Installed:    "Y",
			NotInstalled: " ",
		},
	}
)

// Returns all available border styles
func GlyphStyles() []string {
	return []string{"Plain", "Angled", "Round", "Curly", "Pipes", "ASCII", "Plain-No-X", "Angled-No-X", "Round-No-X", "Curly-No-X", "Pipes-No-X", "ASCII-No-X", "Custom"}
}

// SetGlyphStyle sets a glyph style
func (s *Settings) SetGlyphStyle(style string) error {
	if style == "Custom" {
		var err error
		s.glyphs, err = loadCustomGlyphs()
		if err != nil {
			return err
		}
		return nil
	}
	s.glyphs = glyphStyles[style]
	return nil
}

// Colors exposes our current set of colors
func (s *Settings) Glyphs() Glyphs {
	return s.glyphs
}

func loadCustomGlyphs() (Glyphs, error) {
	colorFile, err := os.UserConfigDir()
	if err != nil {
		return glyphStyles[defaultGlyphStyle], err
	}

	colorFile = path.Join(colorFile, "/pacseek/glyphs.json")

	if _, err := os.Stat(colorFile); errors.Is(err, fs.ErrNotExist) {
		err = createCustomGlyphsFile(colorFile)
		if err != nil {
			return glyphStyles[defaultGlyphStyle], err
		}
	}

	b, err := os.ReadFile(colorFile)
	if err != nil {
		return glyphStyles[defaultGlyphStyle], err
	}

	g := Glyphs{}
	err = json.Unmarshal(b, &g)
	if err != nil {
		return glyphStyles[defaultGlyphStyle], err
	}

	return g, nil
}

// write our color scheme to a json file
func createCustomGlyphsFile(colorFile string) error {
	g := glyphStyles[defaultGlyphStyle]
	b, err := json.MarshalIndent(&g, "", "\t")
	if err != nil {
		return err
	}

	if err = os.WriteFile(colorFile, b, 0644); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
