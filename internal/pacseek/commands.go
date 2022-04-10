package pacseek

import (
	"bufio"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

// installs or removes a package
func (ps *UI) installPackage() {
	row, _ := ps.packages.GetSelection()
	pkg := ps.packages.GetCell(row, 0).Text
	source := ps.packages.GetCell(row, 1).Text
	installed := ps.packages.GetCell(row, 2).Text

	// set command based on source and install status
	command := ps.conf.InstallCommand
	if installed == "Y" {
		command = ps.conf.UninstallCommand
	} else if source == "AUR" && ps.conf.AurUseDifferentCommands && ps.conf.AurInstallCommand != "" {
		command = ps.conf.AurInstallCommand
	}

	// if our command contains {pkg}, replace it with the package name, otherwise concat it
	if strings.Contains(command, "{pkg}") {
		command = strings.Replace(command, "{pkg}", pkg, -1)
	} else {
		command += " " + pkg
	}

	// Here I'm assuming -c is the argument for passing a command to the shell
	// This might not be valid for all of em though.
	args := []string{"-c", command}

	ps.runCommand(ps.shell, args)

	// update package install status
	if isInstalled(ps.alpmHandle, pkg) {
		ps.packages.SetCellSimple(row, 2, "Y")
	} else {
		ps.packages.SetCellSimple(row, 2, "-")
	}
}

// issues "Update command"
func (ps *UI) performUpgrade(aur bool) {
	command := ps.conf.SysUpgradeCommand
	if aur && ps.conf.AurUseDifferentCommands && ps.conf.AurUpgradeCommand != "" {
		command = ps.conf.AurUpgradeCommand
	}

	args := []string{"-c", command}

	ps.runCommand(ps.shell, args)
}

// suspends UI and runs a command in the terminal
func (ps *UI) runCommand(command string, args []string) {
	// suspend gui and run command in terminal
	ps.app.Suspend(func() {

		cmd := exec.Command(command, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// handle SIGINT and forward to the child process
		cmd.Start()
		quit := handleSigint(cmd)
		err := cmd.Wait()
		if err != nil {
			if err.Error() != "signal: interrupt" {
				cmd.Stdout.Write([]byte("\n" + err.Error() + "\nPress ENTER to return to pacseek\n"))
				r := bufio.NewReader(cmd.Stdin)
				r.ReadLine()
			}
		}
		quit <- true
	})
	// we need to reinitialize the alpm handler to get the proper install state
	err := ps.reinitPacmanDbs()
	if err != nil {
		ps.showMessage(err.Error(), true)
		return
	}
}

// handles SIGINT call and passes it to a cmd process
func handleSigint(cmd *exec.Cmd) chan bool {
	quit := make(chan bool, 1)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		select {
		case <-c:
			if cmd != nil {
				cmd.Process.Signal(os.Interrupt)
			}
		case <-quit:
		}
	}()
	return quit
}

// re-initializes the alpm handler
func (ps *UI) reinitPacmanDbs() error {
	err := ps.alpmHandle.Release()
	if err != nil {
		return err
	}
	ps.alpmHandle, err = initPacmanDbs(ps.conf.PacmanDbPath, ps.conf.PacmanConfigPath)
	if err != nil {
		return err
	}
	return nil
}
