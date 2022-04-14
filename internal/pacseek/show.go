package pacseek

import (
	"sort"
	"time"

	"github.com/rivo/tview"
)

// gets packages from repos/AUR and displays them
func (ps *UI) showPackages(text string) {
	go func() {
		ps.locker.Lock()
		defer ps.locker.Unlock()
		defer ps.app.QueueUpdate(func() { ps.showPackageInfo(1, 0) })

		var packages []Package
		packagesCache, foundCache := ps.searchCache.Get(text)

		if !foundCache {
			ps.startSpin()
			defer ps.stopSpin()

			var err error
			packages, err = searchRepos(ps.alpmHandle, text, ps.conf.SearchMode, ps.conf.SearchBy, ps.conf.MaxResults)
			if err != nil {
				ps.app.QueueUpdateDraw(func() {
					ps.showMessage(err.Error(), true)
				})
			}
			if !ps.conf.DisableAur {
				aurPackages, err := searchAur(ps.conf.AurRpcUrl, text, ps.conf.AurTimeout, ps.conf.SearchMode, ps.conf.SearchBy, ps.conf.MaxResults)
				if err != nil {
					ps.app.QueueUpdateDraw(func() {
						ps.showMessage(err.Error(), true)
					})
				}

				for i := 0; i < len(aurPackages); i++ {
					aurPackages[i].IsInstalled = isInstalled(ps.alpmHandle, aurPackages[i].Name)
				}

				packages = append(packages, aurPackages...)
			}

			sort.Slice(packages, func(i, j int) bool {
				return packages[i].Name < packages[j].Name
			})

			if len(packages) == 0 {
				ps.app.QueueUpdateDraw(func() {
					ps.showMessage("No packages found for search-term: "+text, false)
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
					ps.infoCache.Set(pkg.Name, RpcResult{Results: []InfoRecord{pkg}}, time.Duration(ps.conf.CacheExpiry)*time.Minute)
				}
				repoInfos := infoPacman(ps.alpmHandle, repoPkgs)
				for _, pkg := range repoInfos.Results {
					ps.infoCache.Set(pkg.Name, RpcResult{Results: []InfoRecord{pkg}}, time.Duration(ps.conf.CacheExpiry)*time.Minute)
				}

				ps.searchCache.Set(text, packages, time.Duration(ps.conf.CacheExpiry)*time.Minute)
			}
		} else {
			packages = packagesCache.([]Package)
		}

		// draw packages
		ps.app.QueueUpdateDraw(func() {
			if text != ps.search.GetText() {
				return
			}
			ps.drawPackages(packages)
			if ps.right.GetItem(0) == ps.settings {
				ps.right.Clear()
				ps.right.AddItem(ps.details, 0, 1, false)
			}
			r, _ := ps.packages.GetSelection()
			if r > 1 {
				ps.packages.Select(1, 0)
			}
		})
	}()
}

// retrieves package information from repo/AUR and displays them
func (ps *UI) showPackageInfo(row, column int) {
	if row == -1 || row+1 > ps.packages.GetRowCount() {
		return
	}
	ps.details.SetTitle("")
	ps.details.Clear()
	pkg := ps.packages.GetCell(row, 0).Text
	source := ps.packages.GetCell(row, 1).Text

	go func() {
		infoCached, foundCached := ps.infoCache.Get(pkg)
		if source == "AUR" && !foundCached {
			time.Sleep(time.Duration(ps.conf.AurSearchDelay) * time.Millisecond)
		}

		if !ps.isSelected(pkg, true) {
			return
		}
		ps.app.QueueUpdateDraw(func() {
			ps.details.SetTitle(" " + colorTitle + "[::b]" + pkg + " - Retrieving data... ")
		})

		ps.locker.Lock()
		if !foundCached {
			ps.startSpin()
			defer ps.stopSpin()
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
				ps.infoCache.Set(pkg, info, time.Duration(ps.conf.CacheExpiry)*time.Minute)
			}
		} else {
			info = infoCached.(RpcResult)
		}

		// draw results
		ps.app.QueueUpdateDraw(func() {
			if !ps.isSelected(pkg, false) {
				return
			}
			if len(info.Results) != 1 {
				errorMsg := "Package not found"
				if info.Error != "" {
					errorMsg = info.Error
				}
				ps.details.SetTitle(" [red]Error ")
				ps.details.SetCellSimple(0, 0, "[red]s"+errorMsg)
				return
			}
			ps.selectedPackage = &info.Results[0]
			_, _, w, _ := ps.root.GetRect()
			ps.drawPackageInfo(info.Results[0], w)
		})
	}()
}

// shows status bar with error message
func (ps *UI) showMessage(message string, isError bool) {
	txt := message
	if isError {
		txt = "[red]Error: " + message
	}

	ps.status.SetText(txt)
	ps.root.ResizeItem(ps.status, 3, 1)

	go func() {
		ps.messageLocker.Lock()
		defer ps.messageLocker.Unlock()
		time.Sleep(10 * time.Second)
		ps.app.QueueUpdateDraw(func() {
			ps.root.ResizeItem(ps.status, 0, 0)
		})
	}()
}

// show help text
func (ps *UI) showHelp() {
	ps.details.SetTitle(" " + colorTitle + "[::b]Usage ")
	ps.details.Clear().
		SetCellSimple(0, 0, "ENTER: Search; Install or remove a selected package").
		SetCellSimple(1, 0, "TAB / CTRL+Up/Down/Right/Left: Navigate between boxes").
		SetCellSimple(2, 0, "Up/Down: Navigate within package list").
		SetCellSimple(3, 0, "Shift+Left/Right: Change size of package list").
		SetCellSimple(4, 0, "CTRL+S: Open/Close settings").
		SetCellSimple(5, 0, "CTRL+H: Show these instructions").
		SetCellSimple(6, 0, "CTRL+U: Perform sysupgrade").
		SetCellSimple(7, 0, "CTRL+A: Perform AUR upgrade (if configured)").
		SetCellSimple(8, 0, "CTRL+W: Wipe cache").
		SetCellSimple(10, 0, "CTRL+Q: Quit")
}

// show about text
func (ps *UI) showAbout() {
	ps.details.SetTitle(" " + colorTitle + "[::b]About ")
	ps.details.Clear().
		SetCellSimple(0, 0, colorHighlight+"[::b]Version").
		SetCellSimple(0, 1, version).
		SetCellSimple(1, 0, colorHighlight+"[::b]Author").
		SetCellSimple(1, 1, "Mario Oenning").
		SetCellSimple(2, 0, colorHighlight+"[::b]URL").
		SetCellSimple(2, 1, "https://github.com/moson-mo/pacseek")

	pic := `
 .--. 
/ _.-'
\  '-.  ...
 '--' 
`
	s := 3
	for i, l := range tview.WordWrap(pic, 100) {
		ps.details.SetCellSimple(s+i, 0, l)
	}
}

// checks if a given package is currently selected in the package list
func (ps *UI) isSelected(pkg string, queue bool) bool {
	var sel string
	f := func() {
		crow, _ := ps.packages.GetSelection()
		sel = ps.packages.GetCell(crow, 0).Text
	}

	if queue {
		ps.app.QueueUpdate(f)
	} else {
		f()
	}

	return sel == pkg
}
