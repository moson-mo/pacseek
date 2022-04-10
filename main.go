package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/pacseek"
	"github.com/moson-mo/pacseek/internal/util"
)

var helpArgs = []string{"-h", "--h", "-help", "--help", "-?", "--?"}

const helpText = `

For the moment, there is only one argument
that you can pass to pacseek: a search term

For example "pacseek pacseek" will trigger an
immediate search for, you guessed it, "pacseek" :)

Usage / Navigation instructions are displayed
when you start pacseek (without arguments)

Please visit the wiki for further information:

https://github.com/moson-mo/pacseek/wiki

`

func main() {
	if os.Getuid() == 0 {
		fmt.Println("pacseek should not be run as root.")
	}
	term := ""
	if len(os.Args) > 1 {
		term = os.Args[1]
		if util.StringSliceContains(helpArgs, term) {
			printHelp()
			os.Exit(0)
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
	ps, err := pacseek.New(conf)
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
	var name string
	usr, err := user.Current()
	if err != nil {
		name = "my friend"
	} else {
		name = usr.Username
	}
	fmt.Printf("Hello %s,%s", name, helpText)
}
