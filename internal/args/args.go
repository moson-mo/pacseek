package args

import (
	"strings"

	"github.com/pborman/getopt/v2"
)

// Flags struct holds our flag options
type Flags struct {
	Repositories   []string
	SearchTerm     string
	AsciiMode      bool
	MonochromeMode bool
	ShowUpdates    bool
	ShowInstalled  bool
	Help           bool
}

// Parse is parsing our arguments and creates a Flags struct from it
func Parse() Flags {
	repos := getopt.String('r', "", "Limit searching to a comma separated list of repositories")
	term := getopt.String('s', "", "Search-term")
	ascii := getopt.Bool('a', "ASCII mode")
	mono := getopt.Bool('m', "Monochrome mode")
	upd := getopt.Bool('u', "Show updates after startup")
	inst := getopt.Bool('i', "Show installed packages after startup")
	help := getopt.BoolLong("help", 'h', "Show usage / help")
	qhelp := getopt.BoolLong("?", '?', "Show usage / help")

	err := getopt.Getopt(nil)
	if err != nil {
		return Flags{
			Help: true,
		}
	}

	flags := Flags{
		SearchTerm:     *term,
		AsciiMode:      *ascii,
		MonochromeMode: *mono,
		ShowUpdates:    *upd,
		ShowInstalled:  *inst,
	}

	if len(*repos) > 0 {
		flags.Repositories = strings.Split(*repos, ",")
	}

	flags.Help = *help || *qhelp

	if flags.SearchTerm == "" && len(getopt.Args()) > 0 {
		flags.SearchTerm = getopt.Args()[0]
	}

	return flags
}
