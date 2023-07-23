package main

import (
	"fmt"
	"os"

	"github.com/moson-mo/pacseek/internal/args"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/pacseek"
)

const helpText = `
Usage: pacseek [OPTION] [SEARCH-TERM]
	-r 	Limit searching to a comma separated list of repositories
	-s	Search-term
	-a	ASCII mode
	-m	Monochrome mode
	-u	show upgrades after startup
	-i	show installed packages after startup

Examples:

pacseek -r core,extra linux
-> Searches for "linux" in the "core" and "extra" repository

pacseek -ui
-> Show installed packages and a list of upgradeable packages

pacseek pacseek
-> Searches for "pacseek" in all repositories

----------------------------------------------------------------

See also:

Manpage: man pacseek
Wiki:    https://github.com/moson-mo/pacseek/wiki

`

func main() {
	if os.Getuid() == 0 {
		fmt.Println("pacseek should not be run as root.")
		os.Exit(1)
	}

	f := args.Parse()
	if f.Help {
		printHelp()
		os.Exit(0)
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
	ps, err := pacseek.New(conf, f)
	if err != nil {
		printErrorExit("Error during pacseek initialization", err)
	}
	if err = ps.Start(); err != nil {
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
