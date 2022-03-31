package main

import (
	"fmt"
	"os"

	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/pacseek"
)

func main() {
	if os.Getuid() == 0 {
		fmt.Println("pacseek should not be run as root.")
	}
	term := ""
	if len(os.Args) > 1 {
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
