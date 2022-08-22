package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/pacseek"
)

const helpText = `
Usage: pacseek [OPTION]
	-r 	Limit searching to a comma separated list of repositories
	-s	Search term
	-a	ASCII mode
	-m	Monochrome mode
	-u	show upgrades after startup
	-i	show installed packages after startup

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
	asciiMode := false
	monoMode := false
	showUpgrades := false
	showInstalled := false

	r := flag.String("r", "", "Comma separated list of repositories")
	flag.StringVar(&term, "s", "", "Search term")
	flag.BoolVar(&asciiMode, "a", false, "ASCII mode")
	flag.BoolVar(&monoMode, "m", false, "Monochrome mode")
	flag.BoolVar(&showUpgrades, "u", false, "Show upgrades")
	flag.BoolVar(&showInstalled, "i", false, "Show installed")
	flag.Usage = printHelp
	flag.Parse()

	if *r != "" {
		repos = strings.Split(*r, ",")
	}

	if len(os.Args) == 2 && !strings.HasPrefix(os.Args[1], "-") {
		term = os.Args[1]
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
	ps, err := pacseek.New(conf, repos, asciiMode, monoMode)
	if err != nil {
		printErrorExit("Error during pacseek initialization", err)
	}
	if err = ps.Start(term, showUpgrades, showInstalled); err != nil {
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
