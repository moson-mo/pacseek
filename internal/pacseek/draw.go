package pacseek

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/rivo/tview"
)

// draws input fields on settings form
func (ps *UI) drawSettingsFields(disableAur, disableCache, separateAurCommands bool) {
	ps.formSettings.Clear(false)
	mode := 0
	if ps.conf.SearchMode != "StartsWith" {
		mode = 1
	}
	by := 0
	if ps.conf.SearchBy != "Name" {
		by = 1
	}
	cIndex := util.IndexOf(config.ColorSchemes(), ps.conf.ColorScheme)
	bIndex := util.IndexOf(config.BorderStyles(), ps.conf.BorderStyle)

	// handle text/drop-down field changes
	sc := func(txt string) {
		ps.settingsChanged = true
	}

	// input fields
	ps.formSettings.AddDropDown("Color scheme: ", config.ColorSchemes(), cIndex, nil)
	if dd, ok := ps.formSettings.GetFormItemByLabel("Color scheme: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			ps.conf.SetColorScheme(text)
			ps.applyColors()
			if text != ps.conf.ColorScheme {
				ps.settingsChanged = true
			}
		})
	}
	ps.formSettings.AddDropDown("Border style: ", config.BorderStyles(), bIndex, nil)
	if dd, ok := ps.formSettings.GetFormItemByLabel("Border style: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			ps.conf.SetBorderStyle(text)
			if text != ps.conf.BorderStyle {
				ps.settingsChanged = true
			}
		})
	}
	ps.formSettings.AddCheckbox("Disable AUR: ", disableAur, func(checked bool) {
		ps.settingsChanged = true
		ps.drawSettingsFields(checked, disableCache, separateAurCommands)
		ps.app.SetFocus(ps.formSettings)
	})
	if !disableAur {
		ps.formSettings.AddInputField("AUR RPC URL: ", ps.conf.AurRpcUrl, 40, nil, sc).
			AddInputField("AUR timeout (ms): ", strconv.Itoa(ps.conf.AurTimeout), 6, nil, sc).
			AddInputField("AUR search delay (ms): ", strconv.Itoa(ps.conf.AurSearchDelay), 6, nil, sc)
	}
	ps.formSettings.AddCheckbox("Disable Cache: ", disableCache, func(checked bool) {
		ps.settingsChanged = true
		i, _ := ps.formSettings.GetFocusedItemIndex()
		ps.drawSettingsFields(disableAur, checked, separateAurCommands)
		ps.formSettings.SetFocus(i)
		ps.app.SetFocus(ps.formSettings)
	})
	if !disableCache {
		ps.formSettings.AddInputField("Cache expiry (m): ", strconv.Itoa(ps.conf.CacheExpiry), 6, nil, sc)
	}
	ps.formSettings.AddInputField("Max search results: ", strconv.Itoa(ps.conf.MaxResults), 6, nil, sc).
		AddDropDown("Search mode: ", []string{"StartsWith", "Contains"}, mode, func(text string, index int) {
			if text != ps.conf.SearchMode {
				ps.settingsChanged = true
			}
		}).
		AddDropDown("Search by: ", []string{"Name", "Name & Description"}, by, func(text string, index int) {
			if text != ps.conf.SearchBy {
				ps.settingsChanged = true
			}
		}).
		AddInputField("Pacman DB path: ", ps.conf.PacmanDbPath, 40, nil, sc).
		AddInputField("Pacman config path: ", ps.conf.PacmanConfigPath, 40, nil, sc).
		AddCheckbox("Separate AUR commands: ", separateAurCommands, func(checked bool) {
			ps.settingsChanged = true
			i, _ := ps.formSettings.GetFocusedItemIndex()
			ps.drawSettingsFields(disableAur, disableCache, checked)
			ps.formSettings.SetFocus(i)
			ps.app.SetFocus(ps.formSettings)
		})
	if separateAurCommands {
		icom := ps.conf.AurInstallCommand
		if icom == "" {
			icom = ps.conf.InstallCommand
		}
		ucom := ps.conf.AurUpgradeCommand
		if ucom == "" {
			ucom = ps.conf.SysUpgradeCommand
		}
		ps.formSettings.AddInputField("AUR Install command: ", icom, 40, nil, sc).
			AddInputField("AUR Upgrade command: ", ucom, 40, nil, sc)
	}
	ps.formSettings.AddInputField("Install command: ", ps.conf.InstallCommand, 40, nil, sc).
		AddInputField("Upgrade command: ", ps.conf.SysUpgradeCommand, 40, nil, sc).
		AddInputField("Uninstall command: ", ps.conf.UninstallCommand, 40, nil, sc)

	ps.applyDropDownColors()

	// key bindings
	ps.formSettings.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// CTRL + Left navigates to the previous control
		if event.Key() == tcell.KeyLeft && event.Modifiers() == tcell.ModCtrl {
			if ps.prevComponent != nil {
				ps.app.SetFocus(ps.prevComponent)
			} else {
				ps.app.SetFocus(ps.tablePackages)
			}
			return nil
		}
		// Down / Up / TAB for form navigation
		if event.Key() == tcell.KeyDown ||
			event.Key() == tcell.KeyUp ||
			event.Key() == tcell.KeyTab {
			i, b := ps.formSettings.GetFocusedItemIndex()
			if b > -1 {
				i = ps.formSettings.GetFormItemCount() + b
			}
			n := i
			if event.Key() == tcell.KeyUp {
				n-- // move up
			} else {
				n++ // move down
			}
			if i >= 0 && i < ps.formSettings.GetFormItemCount() {
				// drop downs are excluded from Up / Down handling
				if _, ok := ps.formSettings.GetFormItem(i).(*tview.DropDown); ok {
					if event.Key() != tcell.KeyTAB && event.Modifiers() != tcell.ModCtrl {
						return event
					}
				}
			}
			// Leave settings from
			if b == ps.formSettings.GetButtonCount()-1 && event.Key() != tcell.KeyUp {
				ps.app.SetFocus(ps.inputSearch)
				return nil
			}
			if i == 0 && event.Key() == tcell.KeyUp {
				ps.app.SetFocus(ps.tablePackages)
				return nil
			}
			ps.formSettings.SetFocus(n)
			ps.app.SetFocus(ps.formSettings)
			return nil
		}
		return event
	})
}

// draw package information on screen
func (ps *UI) drawPackageInfo(i InfoRecord, width int) {
	ps.tableDetails.Clear().
		SetTitle(" [::b]" + i.Name + " ")
	r := 0
	ln := 0

	fields, order := getDetailFields(i)
	for _, k := range order {
		if v, ok := fields[k]; ok && v != "" {
			if ln == 1 || k == "Last modified" {
				r++
			}
			// split lines if they do not fit on the screen
			w := width - (int(float64(width)*(float64(ps.leftProportion)/10)) + 21) // left box = 40% size, then we use 13 characters for column 0, 2 for padding and 6 for borders
			lines := tview.WordWrap(v, w)
			mr := r
			cell := &tview.TableCell{
				Text:            "[::b]" + k,
				Color:           ps.conf.Colors().Accent,
				BackgroundColor: tcell.ColorBlack,
			}

			if k == " Show PKGBUILD" {
				ps.tableDetails.SetCellSimple(r, 0, "")
				r++
				cell.SetBackgroundColor(ps.conf.Colors().Accent).
					SetTextColor(ps.conf.Colors().SettingsFieldText).
					SetAlign(tview.AlignCenter).
					SetClickedFunc(func() bool {
						ps.runCommand(util.Shell(), []string{"-c", v})
						return true
					})
				ps.tableDetails.SetCell(r, 0, cell)
				break
			}
			ps.tableDetails.SetCell(r, 0, cell)

			for _, l := range lines {
				if mr != r {
					ps.tableDetails.SetCellSimple(r, 0, "") // we need to add some blank content otherwise it looks weird with some terminal configs
				}
				cell := &tview.TableCell{
					Text:            l,
					Color:           tcell.ColorWhite,
					BackgroundColor: tcell.ColorBlack,
				}
				if strings.Contains(k, "URL") {
					cell.SetClickedFunc(func() bool {
						exec.Command("xdg-open", v).Run()
						return true
					})
				}
				ps.tableDetails.SetCell(r, 1, cell)
				r++
			}
			ln++
		}
	}
	ps.tableDetails.ScrollToBeginning()
}

// draw packages on screen
func (ps *UI) drawPackageListContent(packages []Package) {
	ps.tablePackages.Clear()

	// header
	ps.drawPackageListHeader()

	// rows
	for i, pkg := range packages {
		color := ps.conf.Colors().PackagelistSourceRepository
		installed := "-"
		if pkg.IsInstalled {
			installed = "Y"
		}
		if pkg.Source == "AUR" {
			color = ps.conf.Colors().PackagelistSourceAUR
		}

		ps.tablePackages.SetCellSimple(i+1, 0, pkg.Name)
		ps.tablePackages.SetCell(i+1, 1, &tview.TableCell{
			Text:            pkg.Source,
			Color:           color,
			BackgroundColor: tcell.ColorBlack,
		})
		ps.tablePackages.SetCell(i+1, 2, &tview.TableCell{
			Text:            installed,
			Expansion:       1000,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorBlack,
		})
	}
	ps.tablePackages.ScrollToBeginning()
}

// adds header row to package table
func (ps *UI) drawPackageListHeader() {
	columns := []string{"Package", "Source", "Installed"}
	for i, col := range columns {
		col := col
		ps.tablePackages.SetCell(0, i, &tview.TableCell{
			Text:            col,
			NotSelectable:   true,
			Color:           ps.conf.Colors().PackagelistHeader,
			BackgroundColor: tcell.ColorBlack,
			Clicked: func() bool {
				switch col {
				case "Package":
					ps.sortAndRedrawPackageList('N')
				case "Source":
					ps.sortAndRedrawPackageList('S')
				case "Installed":
					ps.sortAndRedrawPackageList('I')
				}
				return true
			},
		})
	}
}

// sorts and redraws the list of packages
func (ps *UI) sortAndRedrawPackageList(runeKey rune) {
	// n - sort by name
	switch runeKey {
	case 'N': // sort by name
		if ps.sortAscending {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				return ps.shownPackages[i].Name > ps.shownPackages[j].Name
			})
		} else {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				return ps.shownPackages[j].Name > ps.shownPackages[i].Name
			})
		}
	case 'S': // sort by source
		if ps.sortAscending {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				if ps.shownPackages[i].Source == ps.shownPackages[j].Source {
					return ps.shownPackages[j].Name > ps.shownPackages[i].Name
				}
				return ps.shownPackages[i].Source > ps.shownPackages[j].Source
			})
		} else {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				if ps.shownPackages[i].Source == ps.shownPackages[j].Source {
					return ps.shownPackages[j].Name > ps.shownPackages[i].Name
				}
				return ps.shownPackages[j].Source > ps.shownPackages[i].Source
			})
		}
	case 'I': // sort by installed state
		if ps.sortAscending {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				if ps.shownPackages[i].IsInstalled == ps.shownPackages[j].IsInstalled {
					return ps.shownPackages[j].Name > ps.shownPackages[i].Name
				}
				return ps.shownPackages[i].IsInstalled
			})
		} else {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				if ps.shownPackages[i].IsInstalled == ps.shownPackages[j].IsInstalled {
					return ps.shownPackages[j].Name > ps.shownPackages[i].Name
				}
				return ps.shownPackages[j].IsInstalled
			})
		}
	case 'M': // sort by last modified date
		if ps.sortAscending {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				return ps.shownPackages[i].LastModified > ps.shownPackages[j].LastModified
			})
		} else {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				return ps.shownPackages[j].LastModified > ps.shownPackages[i].LastModified
			})
		}
	case 'P': // sort by popularity
		if ps.sortAscending {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				if ps.shownPackages[i].Popularity == ps.shownPackages[j].Popularity {
					return ps.shownPackages[j].Name > ps.shownPackages[i].Name
				}
				return ps.shownPackages[i].Popularity > ps.shownPackages[j].Popularity
			})
		} else {
			sort.Slice(ps.shownPackages, func(i, j int) bool {
				if ps.shownPackages[i].Popularity == ps.shownPackages[j].Popularity {
					return ps.shownPackages[j].Name > ps.shownPackages[i].Name
				}
				return ps.shownPackages[j].Popularity > ps.shownPackages[i].Popularity
			})
		}
	}
	ps.sortAscending = !ps.sortAscending
	ps.drawPackageListContent(ps.shownPackages)
	ps.tablePackages.Select(1, 0)
}

// composes a map with fields and values (package information) for our details box
func getDetailFields(i InfoRecord) (map[string]string, []string) {
	order := []string{
		"Description",
		"Version",
		"Provides",
		"Conflicts",
		"Licenses",
		"Maintainer",
		"Dependencies",
		"Required by",
		"URL",
		"Votes",
		"Popularity",
		"Last modified",
		"Flagged out of date",
		"Package URL",
		" Show PKGBUILD", //the space in front is an ugly alignment hack ;)
	}

	fields := map[string]string{}
	fields[order[0]] = "[::b]" + i.Description
	fields[order[1]] = i.Version
	fields[order[2]] = strings.Join(i.Provides, ", ")
	fields[order[3]] = strings.Join(i.Conflicts, ", ")
	fields[order[4]] = strings.Join(i.License, ", ")
	fields[order[5]] = i.Maintainer
	fields[order[6]] = getDependenciesJoined(i)
	fields[order[7]] = strings.Join(i.RequiredBy, ", ")
	fields[order[8]] = i.URL
	if i.Source == "AUR" {
		fields[order[9]] = fmt.Sprintf("%d", i.NumVotes)
		fields[order[10]] = fmt.Sprintf("%f", i.Popularity)
		fields[order[13]] = fmt.Sprintf(UrlAurPackage, i.Name)
	} else if util.SliceContains(getArchRepos(), i.Source) {
		fields[order[13]] = fmt.Sprintf(UrlPackage, i.Source, i.Architecture, i.Name)
	}
	if i.LastModified != 0 {
		fields[order[11]] = time.Unix(int64(i.LastModified), 0).UTC().Format("2006-01-02 - 15:04:05 (UTC)")
	}
	if i.OutOfDate != 0 {
		fields[order[12]] = time.Unix(int64(i.OutOfDate), 0).UTC().Format("[red]2006-01-02 - 15:04:05 (UTC)")
	}
	fields[order[14]] = getPkgbuildCommand(i.Source, i.PackageBase)

	return fields, order
}

// join and format different dependencies as string
func getDependenciesJoined(i InfoRecord) string {
	mdeps := strings.Join(i.MakeDepends, " (make), ")
	if mdeps != "" {
		mdeps += " (make)"
	}
	odeps := strings.Join(i.OptDepends, " (opt), ")
	if odeps != "" {
		odeps += " (opt)"
	}

	deps := strings.Join(i.Depends, ", ")
	if deps != "" && mdeps != "" {
		deps += "\n"
	}
	deps += mdeps
	if deps != "" && odeps != "" {
		deps += "\n"
	}
	deps += odeps
	return deps
}

// updates the "install state" of all packages in cache and package list
func (ps *UI) updateInstalledState() {
	// update cached packages
	sterm := ps.inputSearch.GetText()
	cpkg, exp, found := ps.cacheSearch.GetWithExpiration(sterm)
	if found {
		scpkg := cpkg.([]Package)
		for i := 0; i < len(scpkg); i++ {
			scpkg[i].IsInstalled = isPackageInstalled(ps.alpmHandle, scpkg[i].Name)
		}
		ps.cacheSearch.Set(sterm, scpkg, exp.Sub(time.Now()))
	}

	// update currently shown packages
	for i := 1; i < ps.tablePackages.GetRowCount(); i++ {
		newCell := &tview.TableCell{
			Text:            "-",
			Expansion:       1000,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorBlack,
		}
		if isPackageInstalled(ps.alpmHandle, ps.tablePackages.GetCell(i, 0).Text) {
			newCell.Text = "Y"
		}
		ps.tablePackages.SetCell(i, 2, newCell)
	}
}
