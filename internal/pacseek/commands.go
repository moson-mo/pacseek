package pacseek

import (
	"bufio"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

// installs or removes a package
func (ps *UI) installPackage(pkg InfoRecord, installed bool) {
	// set command based on source and install status
	command := ps.conf.InstallCommand
	if installed {
		command = ps.conf.UninstallCommand
	} else if pkg.Source == "AUR" && ps.conf.AurUseDifferentCommands && ps.conf.AurInstallCommand != "" {
		command = ps.conf.AurInstallCommand
	}

	// if our command contains {pkg}, replace it with the package name, otherwise concat it
	if strings.Contains(command, "{pkg}") {
		command = strings.Replace(command, "{pkg}", pkg.Name, -1)
	} else {
		command += " " + pkg.Name
	}

	// replace optdepends
	command = strings.Replace(command, "{optdepends}", strings.Join(pkg.OptDepends, " "), -1)

	// replace repo
	command = strings.Replace(command, "{repo}", strings.ToLower(pkg.Source), -1)

	// replace {giturl} with AUR url if defined
	if pkg.Source == "AUR" {
		command = strings.Replace(command, "{giturl}", "https://aur.archlinux.org/"+pkg.PackageBase+".git", -1)
		command = strings.Replace(command, "{pkgbase}", pkg.PackageBase, -1)
	}

	// Here I'm assuming -c is the argument for passing a command to the shell
	// This might not be valid for all of em though.
	args := []string{"-c", command}

	ps.runCommand(ps.shell, args...)

	// update package install status
	ps.updateInstalledState()
}

// installs or removes a package
func (ps *UI) installSelectedPackage() {
	if ps.selectedPackage == nil {
		return
	}
	row, _ := ps.tablePackages.GetSelection()
	installed := ps.tablePackages.GetCell(row, 2).Reference == true

	ps.installPackage(*ps.selectedPackage, installed)
}

// issues "Update command"
func (ps *UI) performUpgrade(aur bool) {
	command := ps.conf.SysUpgradeCommand
	if aur && ps.conf.AurUseDifferentCommands && ps.conf.AurUpgradeCommand != "" {
		command = ps.conf.AurUpgradeCommand
	}

	args := []string{"-c", command}

	ps.runCommand(ps.shell, args...)
}

// suspends UI and runs a command in the terminal
func (ps *UI) runCommand(command string, args ...string) {
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
		ps.displayMessage(err.Error(), true)
	}
}

// re-initializes the alpm handler
func (ps *UI) reinitPacmanDbs() error {
	err := ps.alpmHandle.Release()
	if err != nil {
		return err
	}
	ps.alpmHandle, err = initPacmanDbs(ps.conf.PacmanDbPath, ps.conf.PacmanConfigPath, ps.filterRepos)
	if err != nil {
		return err
	}
	return nil
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
