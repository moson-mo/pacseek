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

	var pkgs_names string
	//var pkgs_opts string
	//var pkgs_repos string

	for i := range pkglist {
		pkgs_names += " " + pkglist[i].Pkg.Name
		//pkgs_opts += " " + pkglist[i].pkg.OptDepends
	}

	// if our command contains {pkg}, replace it with the package name, otherwise concat it
	if strings.Contains(command, "{pkg}") {
		command = strings.Replace(command, "{pkg}", pkgs_names, -1)
	} else {
		command += " " + pkgs_names
	}

	// TODO:replace optdepends
	//command = strings.Replace(command, "{optdepends}", strings.Join(pkg.OptDepends, " "), -1)

	// TODO:replace repo
	//command = strings.Replace(command, "{repo}", strings.ToLower(pkg.Source), -1)

	// TODO:replace {giturl} with AUR url if defined
	//if pkg.Source == "AUR" {
	//	command = strings.Replace(command, "{giturl}", "https://aur.archlinux.org/"+pkg.PackageBase+".git", -1)
	//	command = strings.Replace(command, "{pkgbase}", pkg.PackageBase, -1)
	//}

	// Here I'm assuming -c is the argument for passing a command to the shell
	// This might not be valid for all of em though.
	args := []string{"-c", command}

	ps.runCommand(ps.shell, args...)
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
			if (pkglist[i].State | PkgMarked) != (state | PkgMarked) {
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

	// this is for case when user selects package (SPACE) and also presses Enter
	marked := false
	for i := range pkglist {
		if pkglist[i].Pkg.ID == ps.selectedPackage.ID {
			marked = true
			break
		}
	}
	if !marked {
		pkglist = ps.selectPackage(pkglist)
	}

	var pkgs_toinstall []PkgStatus
	var pkgs_touninstall []PkgStatus
	var aur_toinstall []PkgStatus

	for i := range pkglist {
		if pkglist[i].State&PkgInstalled == PkgInstalled {
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
	// clear pkglist since we splitted it to multiple sources
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

// selects a package, returns list(slice) of packages
func (ps *UI) selectPackage(pkglist []PkgStatus) []PkgStatus {
	if ps.selectedPackage == nil {
		return pkglist
	}
	row, _ := ps.tablePackages.GetSelection()
	installed := ps.tablePackages.GetCell(row, 2).Reference.(PkgState) & PkgInstalled

	marked := false
	index := 0

	for i := range pkglist {
		if pkglist[i].Pkg.ID == ps.selectedPackage.ID {
			marked = true
			index = i
			break
		}
	}

	if marked {
		// removes marked item from pkglist thus when getPkgState it returns PkgNone
		var tmp []PkgStatus
		tmp = append(tmp, pkglist[:index]...)
		tmp = append(tmp, pkglist[index+1:]...)
		pkglist = tmp
	} else {
		pkglist = append(pkglist, PkgStatus{*ps.selectedPackage, installed | PkgMarked})
	}
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
