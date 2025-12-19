package pacseek

import (
	"bufio"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

func (ps *UI) runShellInstall(command string, pkglist []PkgStatus) []PkgStatus {
	if pkglist == nil {
		return nil
	}

	// Needs one by one logic
	// NOTE: will need rework if we want to have everything in one batch
	if strings.Contains(command, "{pkg}") ||
		strings.Contains(command, "{optdepends}") ||
		strings.Contains(command, "{repo}") ||
		strings.Contains(command, "{giturl}") {

		for i := range pkglist {
			pkg := pkglist[i].Pkg
			command = strings.Replace(command, "{pkg}", pkg.Name, -1)
			command = strings.Replace(command, "{optdepends}", strings.Join(pkg.OptDepends, " "), -1)
			command = strings.Replace(command, "{repo}", strings.ToLower(pkg.Source), -1)

			if pkg.Source == "AUR" {
				command = strings.Replace(command, "{giturl}", "https://aur.archlinux.org/"+pkg.PackageBase+".git", -1)
				command = strings.Replace(command, "{pkgbase}", pkg.PackageBase, -1)
			}
			// Here I'm assuming -c is the argument for passing a command to the shell
			// This might not be valid for all of em though.
			args := []string{"-c", command}

			ps.runCommand(ps.shell, args...)
		}
	} else {
		for i := range pkglist {
			command += " " + pkglist[i].Pkg.Name
		}
		// Here I'm assuming -c is the argument for passing a command to the shell
		// This might not be valid for all of em though.
		args := []string{"-c", command}

		ps.runCommand(ps.shell, args...)
	}

	ps.cacheSearch.Flush()
	// there is no way of checking if ps.runCommand wasn't aborted
	// it's better to check whether package was installed after aborted
	// if yes then remove it from pkglist
	var tmp []PkgStatus
	var state PkgState
	for range pkglist {
		tmp = nil
		for i := range pkglist {
			if isPackageInstalled(ps.alpmHandle, pkglist[i].Pkg.Name) {
				state = PkgInstalled
			} else {
				state = PkgNone
			}

			// states not equal: means installed has changed thus remove from list
			// 2nd clause is weak against interrupts
			if pkglist[i].State&PkgInstalled != state ||
				pkglist[i].State&PkgReinstall == PkgReinstall {
				pkglistSize := len(pkglist)
				if i+1 == pkglistSize {
					// everything successful return empty array
					return nil
				}
				tmp = append(tmp, pkglist[:i]...)
				// last element
				if i != pkglistSize {
					tmp = append(tmp, pkglist[i+1:]...)
				}
				pkglist = tmp
				break
			}
		}
	}

	return pkglist
}

// installs or removes a packages
// if this command succeeds resulting pkglist will be nil
// we return it cuz it may not succeed e.g. abort install halfway through
func (ps *UI) installSelectedPackages(pkglist []PkgStatus) []PkgStatus {
	if ps.selectedPackage == nil {
		return nil
	}

	// this is for case when user presses Enter on package that wasn't explicitly selected
	// (default actions install when uninstalled and uninstalls when installed)
	row, _ := ps.tablePackages.GetSelection()
	if ps.tablePackages.GetCell(row, 2).Reference.(PkgState) <= PkgInstalled {
		pkglist = ps.selectPackage(pkglist, PkgMarked)
	}

	var pkgs_toinstall []PkgStatus
	var pkgs_touninstall []PkgStatus
	var aur_toinstall []PkgStatus

	for i := range pkglist {
		if pkglist[i].State == PkgInstalled|PkgMarked {
			pkgs_touninstall = append(pkgs_touninstall, pkglist[i])
		} else {
			if pkglist[i].Pkg.Source == "AUR" {
				aur_toinstall = append(aur_toinstall, pkglist[i])
			} else {
				pkgs_toinstall = append(pkgs_toinstall, pkglist[i])
			}
		}
	}

	var command string
	// clear pkglist since we split it to multiple sources
	pkglist = nil

	if pkgs_touninstall != nil {
		command = ps.conf.UninstallCommand
		pkgs_touninstall = ps.runShellInstall(command, pkgs_touninstall)
		pkglist = append(pkglist, pkgs_touninstall...)
	}

	if aur_toinstall != nil && ps.conf.AurUseDifferentCommands && ps.conf.AurInstallCommand != "" {
		command = ps.conf.AurInstallCommand
		aur_toinstall = ps.runShellInstall(command, aur_toinstall)
		pkglist = append(pkglist, aur_toinstall...)
	} else {
		// this will drop down to pkgs_toinstall
		pkgs_toinstall = append(pkgs_toinstall, aur_toinstall...)
	}

	if pkgs_toinstall != nil {
		command = ps.conf.InstallCommand
		pkgs_toinstall = ps.runShellInstall(command, pkgs_toinstall)
		pkglist = append(pkglist, pkgs_toinstall...)
	}

	// update package install status
	ps.updateInstalledState(pkglist)
	return pkglist
}

func (ps *UI) removeDuplicate(pkglist []PkgStatus) []PkgStatus {
	for i := range pkglist {
		if pkglist[i].Pkg.Name == ps.selectedPackage.Name {
			pkglist = append(pkglist[:i], pkglist[i+1:]...)
			break
		}
	}
	return pkglist
}

func (ps *UI) selectPackage(pkglist []PkgStatus, state PkgState) []PkgStatus {
	if ps.selectedPackage == nil {
		return pkglist
	}
	row, _ := ps.tablePackages.GetSelection()
	pkgstate := ps.tablePackages.GetCell(row, 2).Reference.(PkgState)

	if state == PkgReinstall && pkgstate&PkgInstalled != PkgInstalled {
		ps.displayMessage("Cannot mark package for reinstallation, not installed", true)
		goto SkipChecks
	}

	// NOTE: in future if we'll be adding more flags this might need some simplifying
	// performance wise removing element and appending element with new value might not be feasible
	pkglist = ps.removeDuplicate(pkglist)
	if (state == PkgMarked && pkgstate&PkgMarked != PkgMarked) ||
		(state == PkgReinstall && pkgstate&PkgReinstall != PkgReinstall) {
		// mutually exclusive flags
		pkglist = append(
			pkglist,
			PkgStatus{*ps.selectedPackage, pkgstate&PkgInstalled | state},
		)
	}

SkipChecks:
	ps.updateInstalledState(pkglist)

	return pkglist
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
