package args

import (
	"errors"
	"strings"

	"github.com/moson-mo/pacseek/internal/util"
)

// Flags struct holds our flag options
type Flags struct {
	Repositories   []string
	SearchTerm     string
	AsciiMode      bool
	MonochromeMode bool
	ShowUpdates    bool
	ShowInstalled  bool
}

// Parse is parsing our arguments and creates a Flags struct from it
func Parse(args []string) (*Flags, error) {
	flags := &Flags{}
	prevFlag := ""

	// set search term to first argument if it's not a flag
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		flags.SearchTerm = args[0]
	}

	for i, flag := range args {
		// set search term to the last argument if it's not a flag and the previous flag was not a string flag
		if len(args) == i+1 && !strings.HasPrefix(flag, "-") && !util.SliceContains(stringFlags(), prevFlag) {
			flags.SearchTerm = flag
		}
		// if we have a flag
		if strings.HasPrefix(flag, "-") {
			flag = strings.ReplaceAll(flag, "-", "") // strip dashes from flag

			// iterate over characters in flag to allow several options in just one flag, e.g. "-amui"
			for _, r := range flag {
				if !util.SliceContains(possibleFlags(), string(r)) {
					return nil, errors.New("flag not supported: " + flag)
				}
				switch r {
				case 'a':
					flags.AsciiMode = true
				case 'm':
					flags.MonochromeMode = true
				case 'u':
					flags.ShowUpdates = true
				case 'i':
					flags.ShowInstalled = true
				case 's':
					if len(args) <= i+1 {
						return nil, errors.New("search-term argument is missing")
					}
					flags.SearchTerm = args[i+1]
				case 'r':
					if len(args) <= i+1 {
						return nil, errors.New("repositories argument is missing")
					}
					flags.Repositories = strings.Split(args[i+1], ",")
				}
			}
		}
		prevFlag = flag
	}
	flags.SearchTerm = strings.ToLower(flags.SearchTerm)

	return flags, nil
}

func possibleFlags() []string {
	return []string{"s", "r", "a", "m", "u", "i"}
}

func stringFlags() []string {
	return []string{"s", "r"}
}
