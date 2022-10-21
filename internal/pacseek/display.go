package pacseek

import (
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Jguer/go-alpm/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

// gets packages from repos/AUR and displays them
func (ps *UI) displayPackages(text string) {
	var packages []Package

	showFunc := func() {
		ps.shownPackages = packages
		best := bestMatch(text, packages) + 1
		ps.drawPackageListContent(packages, ps.conf.PackageColumnWidth)
		if ps.flexRight.GetItem(0) == ps.formSettings {
			ps.flexRight.Clear()
			ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
		}
		ps.tablePackages.Select(best, 0) // select the best match
	}

	// check cache first
	if packagesCache, found := ps.cacheSearch.Get(text); found {
		packages = packagesCache.([]Package)
		showFunc()
		return
	}

	go func() {
		ps.locker.Lock()
		ps.startSpinner()
		defer func() {
			ps.locker.Unlock()
			ps.stopSpinner()
		}()

		var err error
		var localPackages []Package

		// search repositories
		packages, localPackages, err = searchRepos(ps.alpmHandle, text, ps.conf.SearchMode, ps.conf.SearchBy, ps.conf.MaxResults)
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.displayMessage(err.Error(), true)
			})
		}
		// search AUR
		if !ps.conf.DisableAur {
			aurPackages, err := searchAur(ps.conf.AurRpcUrl, text, ps.conf.AurTimeout, ps.conf.SearchMode, ps.conf.SearchBy, ps.conf.MaxResults)
			if err != nil {
				ps.app.QueueUpdateDraw(func() {
					ps.displayMessage(err.Error(), true)
				})
			}

			for i := 0; i < len(aurPackages); i++ {
				aurPackages[i].IsInstalled = isPackageInstalled(ps.alpmHandle, aurPackages[i].Name)
			}

			// add AUR results to our list
			packages = append(packages, aurPackages...)
		}

		// add local-only (not found in repo not AUR)
		for _, lpkg := range localPackages {
			found := false
			for _, pkg := range packages {
				if pkg.Name == lpkg.Name {
					found = true
					break
				}
			}
			if !found {
				packages = append(packages, lpkg)
			}
		}

		// show message if we couldn't find anything
		if len(packages) == 0 {
			ps.app.QueueUpdateDraw(func() {
				ps.displayMessage("No packages found for search-term: "+text, false)
			})
			return
		}

		// sort list by name
		sort.Slice(packages, func(i, j int) bool {
			return packages[i].Name < packages[j].Name
		})

		// strip down list to our configured maximum
		if len(packages) > ps.conf.MaxResults {
			packages = packages[:ps.conf.MaxResults]
		}

		// get info records and store in cache
		ps.cacheSearchAndPackageInfo(packages, text)

		// draw packages
		ps.app.QueueUpdateDraw(func() {
			showFunc()
		})
	}()
}

// retrieves package info records and stores search results and infos in cache
func (ps *UI) cacheSearchAndPackageInfo(packages []Package, searchTerm string) {
	// get string slices for AUR and repo packages
	aurPkgs := []string{}
	for _, pkg := range packages {
		if pkg.Source == "AUR" {
			aurPkgs = append(aurPkgs, pkg.Name)
		}
	}
	repoPkgs := []string{}
	for _, pkg := range packages {
		if pkg.Source != "AUR" {
			repoPkgs = append(repoPkgs, pkg.Name)
		}
	}

	// get detailed package information for all packages and add to cache
	if !ps.conf.DisableCache {
		aurInfos := infoAur(ps.conf.AurRpcUrl, ps.conf.AurTimeout, aurPkgs...)
		for _, pkg := range aurInfos.Results {
			ps.cacheInfo.Set(pkg.Name+"-"+pkg.Source, pkg, time.Duration(ps.conf.CacheExpiry)*time.Minute)
		}
		repoInfos := infoPacman(ps.alpmHandle, ps.conf.ComputeRequiredBy, repoPkgs...)
		for _, pkg := range repoInfos.Results {
			ps.cacheInfo.Set(pkg.Name+"-"+pkg.Source, pkg, time.Duration(ps.conf.CacheExpiry)*time.Minute)
		}

		ps.cacheSearch.Set(searchTerm, packages, time.Duration(ps.conf.CacheExpiry)*time.Minute)
	}
}

// retrieves package information from repo/AUR and displays them
func (ps *UI) displayPackageInfo(row, column int) {
	if row == -1 || row+1 > ps.tablePackages.GetRowCount() {
		return
	}
	ps.tableDetails.Clear().
		SetTitle("")
	pkg := ps.tablePackages.GetCell(row, 0).Text
	source := ps.tablePackages.GetCell(row, 1).Text

	info := RpcResult{}

	showFunc := func() {
		if !ps.isPackageSelected(pkg, false) {
			return
		}
		if len(info.Results) != 1 {
			errorMsg := "Package not found"
			if info.Error != "" {
				errorMsg = info.Error
			}
			ps.tableDetails.SetTitle(" [red]Error ")
			ps.tableDetails.SetCellSimple(0, 0, "[red]"+errorMsg)
			return
		}
		ps.selectedPackage = &info.Results[0]
		ps.drawPackageInfo(info.Results[0], ps.width)
	}

	if infoCached, found := ps.cacheInfo.Get(pkg + "-" + source); found {
		info.Results = []InfoRecord{infoCached.(InfoRecord)}
		showFunc()
		return
	}

	go func() {
		if source == "AUR" {
			time.Sleep(time.Duration(ps.conf.AurSearchDelay) * time.Millisecond)
		}

		if !ps.isPackageSelected(pkg, true) {
			return
		}

		ps.app.QueueUpdateDraw(func() {
			ps.tableDetails.SetTitle(" [::b]" + pkg + " - Retrieving data... ")
		})

		ps.locker.Lock()
		ps.startSpinner()
		defer func() {
			ps.locker.Unlock()
			ps.stopSpinner()
		}()

		if source == "AUR" {
			info = infoAur(ps.conf.AurRpcUrl, ps.conf.AurTimeout, pkg)
		} else {
			info = infoPacman(ps.alpmHandle, ps.conf.ComputeRequiredBy, pkg)
		}
		if !ps.conf.DisableCache && len(info.Results) == 1 {
			ps.cacheInfo.Set(pkg+"-"+info.Results[0].Source, info.Results[0], time.Duration(ps.conf.CacheExpiry)*time.Minute)
		}

		// draw results
		ps.app.QueueUpdateDraw(func() {
			showFunc()
		})
	}()
}

// displays status bar with error message
func (ps *UI) displayMessage(message string, isError bool) {
	txt := message
	if isError {
		txt = "[red]Error: " + message
	}

	ps.textMessage.SetText(txt)
	ps.flexRoot.ResizeItem(ps.textMessage, 3, 1)

	go func() {
		ps.messageLocker.Lock()
		defer ps.messageLocker.Unlock()
		time.Sleep(5 * time.Second)
		ps.app.QueueUpdateDraw(func() {
			ps.flexRoot.ResizeItem(ps.textMessage, 0, 0)
		})
	}()
}

// displays help text
func (ps *UI) displayHelp() {
	ps.tableDetails.Clear().
		SetTitle(" [::b]" + ps.conf.Glyphs().Help + "Usage ")
	ps.tableDetails.SetCellSimple(0, 0, "ENTER: Search; Install or remove a selected package").
		SetCellSimple(1, 0, "TAB / CTRL+Up/Down/Right/Left: Navigate between boxes").
		SetCellSimple(2, 0, "Up/Down: Navigate within package list").
		SetCellSimple(3, 0, "Shift+Left/Right: Change size of package list").
		SetCellSimple(4, 0, "CTRL+S: Open/Close settings").
		SetCellSimple(5, 0, "CTRL+N: Show these instructions").
		SetCellSimple(6, 0, "CTRL+U: Perform sysupgrade").
		SetCellSimple(7, 0, "CTRL+A: Perform AUR upgrade (if configured)").
		SetCellSimple(8, 0, "CTRL+W: Wipe cache").
		SetCellSimple(9, 0, "CTRL+P: Show PKGBUILD for selected package").
		SetCellSimple(10, 0, "CTRL+O: Open URL for selected package").
		SetCellSimple(11, 0, "CTRL+G: Show list of upgradeable packages").
		SetCellSimple(12, 0, "CTRL+L: Show list of all installed packages").
		SetCellSimple(14, 0, "CTRL+Q / ESC: Quit").
		SetCell(16, 0, &tview.TableCell{
			Text:            "For detailed instructions, please check the man page or visit the [::b]Wiki",
			Color:           tcell.ColorWhite,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
			Clicked: func() bool {
				exec.Command("xdg-open", "https://github.com/moson-mo/pacseek/wiki/Usage").Start()
				return true
			},
		})
}

// displays about text
func (ps *UI) displayAbout() {
	ps.tableDetails.SetTitle(" [::b]About ")
	ps.tableDetails.Clear().
		SetCell(0, 0, &tview.TableCell{
			Text:            "[::b]Version",
			Color:           ps.conf.Colors().Accent,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
		}).
		SetCellSimple(0, 1, version).
		SetCell(1, 0, &tview.TableCell{
			Text:            "[::b]Author",
			Color:           ps.conf.Colors().Accent,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
		}).
		SetCellSimple(1, 1, "Mario Oenning").
		SetCell(2, 0, &tview.TableCell{
			Text:            "[::b]URL",
			Color:           ps.conf.Colors().Accent,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
		}).
		SetCellSimple(2, 1, "https://github.com/moson-mo/pacseek")

	pic := `
 .--. 
/ _.-'
\  '-.  ...
 '--' 
`
	s := 3
	for i, l := range tview.WordWrap(pic, 100) {
		ps.tableDetails.SetCellSimple(s+i, 0, l)
	}
}

// displays PKGBUILD file
func (ps *UI) displayPkgbuild() {
	if ps.selectedPackage == nil {
		return
	}
	pkg := *ps.selectedPackage

	ps.textPkgbuild.Clear().
		SetTitle(" [::b]Loading PKGBUILD... ")
	ps.flexRight.Clear().
		AddItem(ps.textPkgbuild, 0, 1, true)
	ps.app.SetFocus(ps.textPkgbuild)

	// check cache first
	if contentCached, found := ps.cachePkgbuild.Get(pkg.PackageBase); found {
		content := contentCached.(string)
		ps.drawPkgbuild(content, pkg.Name)
		return
	}

	go func() {
		ps.locker.Lock()
		ps.startSpinner()
		defer func() {
			ps.locker.Unlock()
			ps.stopSpinner()
		}()

		content, err := getPkgbuildContent(getPkgbuildUrl(pkg.Source, pkg.PackageBase))
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.textPkgbuild.SetTitle(" [::b]Error loading PKGBUILD ")
				ps.textPkgbuild.SetText(err.Error())
			})
			return
		}
		if !ps.conf.DisableCache {
			ps.cachePkgbuild.Set(pkg.PackageBase, content, time.Duration(ps.conf.CacheExpiry)*time.Minute)
		}
		ps.app.QueueUpdateDraw(func() {
			ps.drawPkgbuild(content, pkg.Name)
		})
	}()
}

// checks if a given package is currently selected in the package list
func (ps *UI) isPackageSelected(pkg string, queue bool) bool {
	var sel string
	f := func() {
		crow, _ := ps.tablePackages.GetSelection()
		sel = ps.tablePackages.GetCell(crow, 0).Text
	}

	if queue {
		ps.app.QueueUpdate(f)
	} else {
		f()
	}

	return sel == pkg
}

// displays a list of updatable packages
func (ps *UI) displayUpgradable() {
	ps.tableDetails.Clear().
		SetTitle(" [::b]Searching for updates... ")

	// check cache first
	if cached, found := ps.cacheInfo.Get("#upgrades#"); found {
		foundUp := cached.([]InfoRecord)
		ps.drawUpgradable(foundUp, true)
		return
	}

	go func() {
		ps.locker.Lock()
		ps.startSpinner()
		defer ps.stopSpinner()
		defer ps.locker.Unlock()

		h, err := syncToTempDB(ps.conf.PacmanConfigPath, ps.filterRepos)
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.tableDetails.SetTitle(" [::b]Error ")

				lines := strings.Split(err.Error(), "\n")
				for i, line := range lines {
					ps.tableDetails.SetCell(i+1, 0, &tview.TableCell{
						Text:            line,
						Color:           tcell.ColorRed,
						BackgroundColor: ps.conf.Colors().DefaultBackground,
					})
				}
				ps.displayMessage("Failed to sync temporary DB's", true)
			})
			return
		}

		up, nf := getUpgradable(h, ps.conf.ComputeRequiredBy)
		aurPkgs := infoAur(ps.conf.AurRpcUrl, ps.conf.AurTimeout, nf...)
		for _, aurPkg := range aurPkgs.Results {
			for i := 0; i < len(up); i++ {
				if up[i].Source == "local" && up[i].Name == aurPkg.Name {
					if alpm.VerCmp(aurPkg.Version, up[i].LocalVersion) > 0 {
						up[i].Description = aurPkg.Description
						up[i].Version = aurPkg.Version
						up[i].Source = "AUR"
					}
				}
			}
		}
		foundUp := []InfoRecord{}
		for _, pkg := range up {
			if pkg.Version != pkg.LocalVersion {
				foundUp = append(foundUp, pkg)
			}
		}
		if !ps.conf.DisableCache {
			ps.cacheInfo.Set("#upgrades#", foundUp, time.Duration(ps.conf.CacheExpiry)*time.Minute)
		}
		ps.app.QueueUpdateDraw(func() {
			ps.drawUpgradable(foundUp, false)
		})
	}()
}

// displays list of installed packages
func (ps *UI) displayInstalled(displayUpdatesAfter bool) {
	ps.tablePackages.Clear().
		SetCellSimple(0, 0, "Generating list, please wait...")

	// search cache
	if installedCached, found := ps.cacheSearch.Get("#installed#"); found {
		packages := installedCached.([]Package)
		ps.shownPackages = packages
		ps.drawPackageListContent(packages, ps.conf.PackageColumnWidth)
		ps.tablePackages.Select(1, 0)
		return
	}

	go func() {
		ps.locker.Lock()
		ps.startSpinner()
		defer func() {
			ps.locker.Unlock()
			ps.stopSpinner()
		}()

		in, nf := getInstalled(ps.alpmHandle, ps.conf.ComputeRequiredBy)
		aurPkgs := infoAur(ps.conf.AurRpcUrl, ps.conf.AurTimeout, nf...).Results
		for _, aurPkg := range aurPkgs {
			for i := 0; i < len(in); i++ {
				if in[i].Source == "local" && in[i].Name == aurPkg.Name {
					in[i].Description = aurPkg.Description
					in[i].Version = aurPkg.Version
					in[i].Source = "AUR"
					in[i].Popularity = aurPkg.Popularity
					in[i].NumVotes = aurPkg.NumVotes
				}
			}
		}

		packages := []Package{}
		for _, pkg := range in {
			packages = append(packages, Package{
				Name:         pkg.Name,
				Source:       pkg.Source,
				IsInstalled:  true,
				LastModified: pkg.LastModified,
				Popularity:   pkg.Popularity,
			})
			if !ps.conf.DisableCache {
				ps.cacheInfo.Set(pkg.Name+"-"+pkg.Source, pkg, time.Duration(ps.conf.CacheExpiry)*time.Minute)
			}
		}
		if !ps.conf.DisableCache {
			ps.cacheSearch.Set("#installed#", packages, time.Duration(ps.conf.CacheExpiry)*time.Minute)
		}
		ps.shownPackages = packages
		ps.app.QueueUpdateDraw(func() {
			ps.drawPackageListContent(packages, ps.conf.PackageColumnWidth)
			if displayUpdatesAfter {
				ps.displayUpgradable()
			} else {
				ps.tablePackages.Select(1, 0)
			}
		})
	}()
}

// returns index of best matching package name
func bestMatch(text string, packages []Package) int {
	// rank packages and save index of closest one to search term
	bestMatch := 0
	prevRank := 9999
	for i := 0; i < len(packages); i++ {
		rank := fuzzy.RankMatch(text, packages[i].Name)
		if rank < prevRank && rank != -1 {
			bestMatch = i
			prevRank = rank
		}
		if rank == 0 {
			break
		}
	}
	return bestMatch
}
