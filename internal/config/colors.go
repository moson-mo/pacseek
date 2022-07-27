package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
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
			PackagelistSourceRepository: tcell.ColorGreen,
			PackagelistSourceAUR:        tcell.NewHexColor(0x1793d1),
			PackagelistHeader:           tcell.ColorYellow,
			SettingsFieldBackground:     tcell.NewHexColor(0x0564A0),
			SettingsFieldText:           tcell.ColorWhite,
			SettingsFieldLabel:          tcell.ColorYellow,
			SettingsDropdownNotSelected: tcell.ColorDarkBlue,
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
			SettingsDropdownNotSelected: tcell.ColorDarkBlue,
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

	b, err := ioutil.ReadFile(colorFile)
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

	if err = ioutil.WriteFile(colorFile, b, 0644); err != nil {
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
	return []string{"Arch Linux", "Endeavour OS", "Monochrome", "Custom"}
}
