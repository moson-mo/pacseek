package pacseek

import (
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

// gets packages from repos/AUR and displays them
func (ps *UI) displayPackages(text string) {
	go func() {
		ps.locker.Lock()
		defer ps.locker.Unlock()
		defer ps.app.QueueUpdate(func() { ps.displayPackageInfo(1, 0) })

		var packages []Package
		packagesCache, foundCache := ps.cacheSearch.Get(text)

		if !foundCache {
			ps.startSpinner()
			defer ps.stopSpinner()

			var err error
			packages, err = searchRepos(ps.alpmHandle, text, ps.conf.SearchMode, ps.conf.SearchBy, ps.conf.MaxResults, false)
			if err != nil {
				ps.app.QueueUpdateDraw(func() {
					ps.displayMessage(err.Error(), true)
				})
			}
			localPackages, err := searchRepos(ps.alpmHandle, text, ps.conf.SearchMode, ps.conf.SearchBy, ps.conf.MaxResults, true)
			if err != nil {
				ps.app.QueueUpdateDraw(func() {
					ps.displayMessage(err.Error(), true)
				})
			}
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

				packages = append(packages, aurPackages...)
			}
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

			sort.Slice(packages, func(i, j int) bool {
				return packages[i].Name < packages[j].Name
			})

			if len(packages) == 0 {
				ps.app.QueueUpdateDraw(func() {
					ps.displayMessage("No packages found for search-term: "+text, false)
				})
			}
			if len(packages) > ps.conf.MaxResults {
				packages = packages[:ps.conf.MaxResults]
			}

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

			if !ps.conf.DisableCache {
				aurInfos := infoAur(ps.conf.AurRpcUrl, aurPkgs, ps.conf.AurTimeout)
				for _, pkg := range aurInfos.Results {
					ps.cacheInfo.Set(pkg.Name, RpcResult{Results: []InfoRecord{pkg}}, time.Duration(ps.conf.CacheExpiry)*time.Minute)
				}
				repoInfos := infoPacman(ps.alpmHandle, repoPkgs)
				for _, pkg := range repoInfos.Results {
					ps.cacheInfo.Set(pkg.Name, RpcResult{Results: []InfoRecord{pkg}}, time.Duration(ps.conf.CacheExpiry)*time.Minute)
				}

				ps.cacheSearch.Set(text, packages, time.Duration(ps.conf.CacheExpiry)*time.Minute)
			}
		} else {
			packages = packagesCache.([]Package)
		}
		ps.shownPackages = packages

		best := bestMatch(text, packages) + 1

		// draw packages
		ps.app.QueueUpdateDraw(func() {
			if text != ps.inputSearch.GetText() {
				return
			}
			ps.drawPackageListContent(packages)
			if ps.flexRight.GetItem(0) == ps.formSettings {
				ps.flexRight.Clear()
				ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
			}
			ps.tablePackages.Select(best, 0) // select the best match
		})
	}()
}

// retrieves package information from repo/AUR and displays them
func (ps *UI) displayPackageInfo(row, column int) {
	if row == -1 || row+1 > ps.tablePackages.GetRowCount() {
		return
	}
	ps.tableDetails.SetTitle("")
	ps.tableDetails.Clear()
	pkg := ps.tablePackages.GetCell(row, 0).Text
	source := ps.tablePackages.GetCell(row, 1).Text

	go func() {
		infoCached, foundCached := ps.cacheInfo.Get(pkg)
		if source == "AUR" && !foundCached {
			time.Sleep(time.Duration(ps.conf.AurSearchDelay) * time.Millisecond)
		}

		if !ps.isPackageSelected(pkg, true) {
			return
		}
		ps.app.QueueUpdateDraw(func() {
			ps.tableDetails.SetTitle(" [::b]" + pkg + " - Retrieving data... ")
		})

		ps.locker.Lock()
		if !foundCached {
			ps.startSpinner()
			defer ps.stopSpinner()
		}
		defer ps.locker.Unlock()

		var info RpcResult
		if !foundCached {
			if source == "AUR" {
				info = infoAur(ps.conf.AurRpcUrl, []string{pkg}, ps.conf.AurTimeout)
			} else {
				info = infoPacman(ps.alpmHandle, []string{pkg})
			}
			if !ps.conf.DisableCache {
				ps.cacheInfo.Set(pkg, info, time.Duration(ps.conf.CacheExpiry)*time.Minute)
			}
		} else {
			info = infoCached.(RpcResult)
		}

		// draw results
		ps.app.QueueUpdateDraw(func() {
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
			_, _, w, _ := ps.flexRoot.GetRect()
			ps.drawPackageInfo(info.Results[0], w)
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
		time.Sleep(10 * time.Second)
		ps.app.QueueUpdateDraw(func() {
			ps.flexRoot.ResizeItem(ps.textMessage, 0, 0)
		})
	}()
}

// displays help text
func (ps *UI) displayHelp() {
	ps.tableDetails.Clear().
		SetTitle(" [::b]Usage ")
	ps.tableDetails.SetCellSimple(0, 0, "ENTER: Search; Install or remove a selected package").
		SetCellSimple(1, 0, "TAB / CTRL+Up/Down/Right/Left: Navigate between boxes").
		SetCellSimple(2, 0, "Up/Down: Navigate within package list").
		SetCellSimple(3, 0, "Shift+Left/Right: Change size of package list").
		SetCellSimple(4, 0, "CTRL+S: Open/Close settings").
		SetCellSimple(5, 0, "CTRL+N: Show these instructions").
		SetCellSimple(6, 0, "CTRL+U: Perform sysupgrade").
		SetCellSimple(7, 0, "CTRL+A: Perform AUR upgrade (if configured)").
		SetCellSimple(8, 0, "CTRL+W: Wipe cache").
		SetCellSimple(10, 0, "CTRL+Q / ESC: Quit")
}

// displays about text
func (ps *UI) displayAbout() {
	ps.tableDetails.SetTitle(" [::b]About ")
	ps.tableDetails.Clear().
		SetCell(0, 0, &tview.TableCell{
			Text:            "[::b]Version",
			Color:           ps.conf.Colors().Accent,
			BackgroundColor: tcell.ColorBlack,
		}).
		SetCellSimple(0, 1, version).
		SetCell(1, 0, &tview.TableCell{
			Text:            "[::b]Author",
			Color:           ps.conf.Colors().Accent,
			BackgroundColor: tcell.ColorBlack,
		}).
		SetCellSimple(1, 1, "Mario Oenning").
		SetCell(2, 0, &tview.TableCell{
			Text:            "[::b]URL",
			Color:           ps.conf.Colors().Accent,
			BackgroundColor: tcell.ColorBlack,
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

// returns index of best matching package name
func bestMatch(text string, packages []Package) int {
	// rank packages and save index of closest one to search term
	bestMatch := 0
	prevRank := 9999
	for i := 0; i < len(packages); i++ {
		rank := fuzzy.RankMatch(text, packages[i].Name)
		if rank < prevRank {
			bestMatch = i
			prevRank = rank
		}
		if rank == 0 {
			break
		}
	}
	return bestMatch
}
