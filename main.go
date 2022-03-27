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
		os.Exit(1)
	}
	conf, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) && conf != nil {
			err = conf.Save()
			if err != nil {
				printError("Error saving configuration file", err)
				os.Exit(1)
			}
		} else {
			printError("Error loading configuration file", err)
			os.Exit(1)
		}
	}
	ps, err := pacseek.New(conf)
	if err != nil {
		printError("Error during pacseek initialization", err)
		os.Exit(1)
	}
	if err = ps.Start(); err != nil {
		printError("Error starting pacseek", err)
		os.Exit(1)
	}
}

func printError(message string, err error) {
	fmt.Printf("%s:\n\n%s\n", message, err.Error())
}
