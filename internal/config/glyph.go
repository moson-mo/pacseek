package config

type Glyphs struct {
	Package      string
	Installed    string
	NotInstalled string
	PrefixState  string
	SuffixState  string
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
		},
		"Angled": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "[",
			SuffixState:  "]",
		},
		"Round": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "(",
			SuffixState:  ")",
		},
		"Curly": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "{",
			SuffixState:  "}",
		},
		"Pipes": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: "✗",
			PrefixState:  "|",
			SuffixState:  "|",
		},
		"ASCII": {
			Package:      "",
			Installed:    "Y",
			NotInstalled: "-",
			PrefixState:  "",
			SuffixState:  "",
		},
		"Plain-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
		},
		"Angled-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "[",
			SuffixState:  "]",
		},
		"Round-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "(",
			SuffixState:  ")",
		},
		"Curly-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "{",
			SuffixState:  "}",
		},
		"Pipes-No-X": {
			Package:      "📦 ",
			Installed:    "✔",
			NotInstalled: " ",
			PrefixState:  "|",
			SuffixState:  "|",
		},
		"ASCII-No-X": {
			Package:      "",
			Installed:    "Y",
			NotInstalled: " ",
			PrefixState:  "",
			SuffixState:  "",
		},
	}
)

// Returns all available border styles
func GlyphStyles() []string {
	return []string{"Plain", "Angled", "Round", "Curly", "Pipes", "ASCII", "Plain-No-X", "Angled-No-X", "Round-No-X", "Curly-No-X", "Pipes-No-X", "ASCII-No-X"}
}

// SetGlyphStyle sets a glyph style
func (s *Settings) SetGlyphStyle(style string) {
	s.glyphs = glyphStyles[style]
}

// Colors exposes our current set of colors
func (s *Settings) Glyphs() Glyphs {
	return s.glyphs
}
