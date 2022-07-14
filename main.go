package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/pacseek"
	"github.com/moson-mo/pacseek/internal/util"
)

var helpArgs = []string{"-h", "--h", "-help", "--help", "-?", "--?"}

const helpText = `
Usage: pacseek [OPTION]
	-r 	Limit searching to a comma separated list of repositories
	-s	Search term

Examples:

pacseek -r core -s linux
-> Searches for "linux" in the "core" repository

pacseek -r core,extra -s linux
-> Searches for "linux" in the "core" and "extra" repository

pacseek -r core,extra
-> Does not start an immediate search,
   but limits searching to "core" and "extra" repositories

Alternatively you can use "pacseek [SEARCH_TERM]":
pacseek pacseek
-> Searches for "pacseek" in all repositories

----------------------------------------------------------------

UI usage / navigation instructions are displayed
when you start pacseek (without specifying a search term)

Please visit the wiki for further information:

https://github.com/moson-mo/pacseek/wiki

`

func main() {
	if os.Getuid() == 0 {
		fmt.Println("pacseek should not be run as root.")
	}
	term := ""
	repos := []string{}
	if len(os.Args) == 2 && !util.StringSliceContains(helpArgs, os.Args[1]) {
		term = os.Args[1]
	} else if len(os.Args) > 1 && util.StringSliceContains(helpArgs, os.Args[1]) {
		printHelp()
		os.Exit(0)
	} else if len(os.Args) > 2 {
		r := flag.String("r", "", "Comma separated list of repositories")
		s := flag.String("s", "", "Search term")
		flag.Usage = printHelp
		flag.Parse()

		term = *s
		if *r != "" {
			repos = strings.Split(*r, ",")
		}
	}

	conf, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) && conf != nil {
			err = conf.Save()
			if err != nil {
				printErrorExit("Error saving configuration file", err)
			}
		} else {
			printErrorExit("Error loading configuration file", err)
		}
	}
	ps, err := pacseek.New(conf, repos)
	if err != nil {
		printErrorExit("Error during pacseek initialization", err)
	}
	if err = ps.Start(term); err != nil {
		printErrorExit("Error starting pacseek", err)
	}
}

func printErrorExit(message string, err error) {
	fmt.Printf("%s:\n\n%s\n", message, err.Error())
	os.Exit(1)
}

func printHelp() {
	fmt.Printf("%s", helpText)
}
