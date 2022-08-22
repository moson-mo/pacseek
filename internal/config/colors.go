package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

// Colors contains all colors of our color scheme
type Colors struct {
	Accent                      tcell.Color
	Title                       tcell.Color
	SearchBar                   tcell.Color
	PackagelistSourceRepository tcell.Color
	PackagelistSourceAUR        tcell.Color
	PackagelistHeader           tcell.Color
	SettingsFieldBackground     tcell.Color
	SettingsFieldText           tcell.Color
	SettingsFieldLabel          tcell.Color
	SettingsDropdownNotSelected tcell.Color
	StylePKGBUILD               string
	colorParsingError           error
}

// default scheme
const (
	defaultColorScheme = "Arch Linux"
)

// color scheme definitions
var (
	colorSchemes = map[string]Colors{
		"Arch Linux": {
			Accent:                      tcell.NewHexColor(0x1793d1),
			Title:                       tcell.NewHexColor(0x00dfff),
			SearchBar:                   tcell.NewHexColor(0x0564A0),
			PackagelistSourceRepository: tcell.NewHexColor(0x00b000),
			PackagelistSourceAUR:        tcell.NewHexColor(0x1793d1),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0x0564A0),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.NewHexColor(0x0564A0),
			StylePKGBUILD:               "dracula",
		},
		"Endeavour OS": {
			Accent:                      tcell.NewHexColor(0x7f7fff),
			Title:                       tcell.NewHexColor(0x7f3fbf),
			SearchBar:                   tcell.NewHexColor(0x7f3fbf),
			PackagelistSourceRepository: tcell.NewHexColor(0xff7f7f),
			PackagelistSourceAUR:        tcell.NewHexColor(0x7f3fbf),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0x7f3fbf),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.NewHexColor(0x7f3fbf),
			StylePKGBUILD:               "friendly",
		},
		"Red": {
			Accent:                      tcell.NewHexColor(0xcc3300),
			Title:                       tcell.NewHexColor(0xff3300),
			SearchBar:                   tcell.NewHexColor(0xcc3300),
			PackagelistSourceRepository: tcell.NewHexColor(0xff9900),
			PackagelistSourceAUR:        tcell.NewHexColor(0xcc3300),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0xcc3300),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.NewHexColor(0xcc3300),
			StylePKGBUILD:               "autumn",
		},
		"Green": {
			Accent:                      tcell.NewHexColor(0x33cc33),
			Title:                       tcell.NewHexColor(0x00ff00),
			SearchBar:                   tcell.NewHexColor(0x009933),
			PackagelistSourceRepository: tcell.NewHexColor(0xffff00),
			PackagelistSourceAUR:        tcell.NewHexColor(0x009933),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0x009933),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.NewHexColor(0x009933),
			StylePKGBUILD:               "igor",
		},
		"Blue": {
			Accent:                      tcell.NewHexColor(0x0066ff),
			Title:                       tcell.NewHexColor(0x0099ff),
			SearchBar:                   tcell.NewHexColor(0x0066ff),
			PackagelistSourceRepository: tcell.NewHexColor(0x00ccff),
			PackagelistSourceAUR:        tcell.NewHexColor(0x0066ff),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0x0066ff),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.NewHexColor(0x0066ff),
			StylePKGBUILD:               "solarized-dark256",
		},
		"Orange": {
			Accent:                      tcell.NewHexColor(0xcc7a00),
			Title:                       tcell.NewHexColor(0xffcc00),
			SearchBar:                   tcell.NewHexColor(0xcc7a00),
			PackagelistSourceRepository: tcell.NewHexColor(0xff6600),
			PackagelistSourceAUR:        tcell.NewHexColor(0xcc7a00),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0xcc7a00),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.NewHexColor(0xcc7a00),
			StylePKGBUILD:               "fruity",
		},
		"Monochrome": {
			Accent:                      tcell.ColorWhite,
			Title:                       tcell.ColorWhite,
			SearchBar:                   tcell.ColorBlack,
			PackagelistSourceRepository: tcell.ColorWhite,
			PackagelistSourceAUR:        tcell.ColorWhite,
			PackagelistHeader:           tcell.ColorWhite,
			SettingsFieldBackground:     tcell.ColorBlack,
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorWhite,
			SettingsDropdownNotSelected: tcell.ColorBlack,
			StylePKGBUILD:               "bw",
		},
	}
)

// SetColorScheme applies a color scheme
func (s *Settings) SetColorScheme(scheme string) error {
	//s.ColorScheme = scheme
	if scheme == "Custom" {
		var err error
		s.colors, err = loadCustomColors()
		if err != nil {
			return err
		}
		return nil
	}
	s.colors = colorSchemes[scheme]
	return nil
}

// loads custom colors from file
func loadCustomColors() (Colors, error) {
	colorFile, err := os.UserConfigDir()
	if err != nil {
		return colorSchemes[defaultColorScheme], err
	}

	colorFile = path.Join(colorFile, "/pacseek/colors.json")

	if _, err := os.Stat(colorFile); errors.Is(err, fs.ErrNotExist) {
		err = createCustomColorsFile(colorFile)
		if err != nil {
			return colorSchemes[defaultColorScheme], err
		}
	}

	b, err := os.ReadFile(colorFile)
	if err != nil {
		return colorSchemes[defaultColorScheme], err
	}

	c := Colors{}
	err = c.unmarshalJSON(b)
	if err != nil {
		return colorSchemes[defaultColorScheme], err
	}
	if c.colorParsingError != nil {
		return colorSchemes[defaultColorScheme], c.colorParsingError
	}

	return c, nil
}

// write our color scheme to a json file
func createCustomColorsFile(colorFile string) error {
	c := colorSchemes[defaultColorScheme]
	b, err := c.marshalJSON()
	if err != nil {
		return err
	}

	if err = os.WriteFile(colorFile, b, 0644); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// custom JSON marshalling for our colors
func (c *Colors) marshalJSON() ([]byte, error) {
	return json.MarshalIndent(&struct {
		Accent                      string
		Title                       string
		SearchBar                   string
		PackagelistSourceRepository string
		PackagelistSourceAUR        string
		PackagelistHeader           string
		SettingsFieldBackground     string
		SettingsFieldText           string
		SettingsFieldLabel          string
		SettingsDropdownNotSelected string
		StylePKGBUILD               string
		Comments                    string
	}{
		Accent:                      fmt.Sprintf("%06x", c.Accent.Hex()),
		Title:                       fmt.Sprintf("%06x", c.Title.Hex()),
		SearchBar:                   fmt.Sprintf("%06x", c.SearchBar.Hex()),
		PackagelistSourceRepository: fmt.Sprintf("%06x", c.PackagelistSourceRepository.Hex()),
		PackagelistSourceAUR:        fmt.Sprintf("%06x", c.PackagelistSourceAUR.Hex()),
		PackagelistHeader:           fmt.Sprintf("%06x", c.PackagelistHeader.Hex()),
		SettingsFieldBackground:     fmt.Sprintf("%06x", c.SettingsFieldBackground.Hex()),
		SettingsFieldText:           fmt.Sprintf("%06x", c.SettingsFieldText.Hex()),
		SettingsFieldLabel:          fmt.Sprintf("%06x", c.SettingsFieldLabel.Hex()),
		SettingsDropdownNotSelected: fmt.Sprintf("%06x", c.SettingsDropdownNotSelected.Hex()),
		StylePKGBUILD:               c.StylePKGBUILD,
		Comments:                    "Examples for StylePKGBUILD can be found here: https://xyproto.github.io/splash/docs/all.html",
	}, "", "\t")
}

// custom JSON unmarshalling for our colors
func (c *Colors) unmarshalJSON(data []byte) error {
	d := &struct {
		Accent                      string
		Title                       string
		SearchBar                   string
		PackagelistSourceRepository string
		PackagelistSourceAUR        string
		PackagelistHeader           string
		SettingsFieldBackground     string
		SettingsFieldText           string
		SettingsFieldLabel          string
		SettingsDropdownNotSelected string
		StylePKGBUILD               string
	}{}
	err := json.Unmarshal(data, d)
	if err != nil {
		return err
	}

	c.Accent = c.colorFromHexString(d.Accent)
	c.Title = c.colorFromHexString(d.Title)
	c.SearchBar = c.colorFromHexString(d.SearchBar)
	c.PackagelistSourceRepository = c.colorFromHexString(d.PackagelistSourceRepository)
	c.PackagelistSourceAUR = c.colorFromHexString(d.PackagelistSourceAUR)
	c.PackagelistHeader = c.colorFromHexString(d.PackagelistHeader)
	c.SettingsFieldBackground = c.colorFromHexString(d.SettingsFieldBackground)
	c.SettingsFieldText = c.colorFromHexString(d.SettingsFieldText)
	c.SettingsFieldLabel = c.colorFromHexString(d.SettingsFieldLabel)
	c.SettingsDropdownNotSelected = c.colorFromHexString(d.SettingsDropdownNotSelected)
	c.StylePKGBUILD = d.StylePKGBUILD
	return nil
}

// convert (color) hex string to tcell.Color
func (c *Colors) colorFromHexString(val string) tcell.Color {
	v, err := strconv.ParseInt(val, 16, 32)
	if err != nil {
		c.colorParsingError = err
		return tcell.ColorRed
	}
	return tcell.NewHexColor(int32(v))
}

// Colors exposes our current set of colors
func (s *Settings) Colors() Colors {
	return s.colors
}

// Returns all available color schemes
func ColorSchemes() []string {
	return []string{"Arch Linux", "Endeavour OS", "Red", "Green", "Blue", "Orange", "Monochrome", "Custom"}
}
