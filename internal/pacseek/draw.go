package pacseek

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/rivo/tview"
)

// draws input fields on settings form
func (ps *UI) drawSettingsFields(disableAur, disableCache, separateAurCommands bool) {
	ps.settings.Clear(false)
	mode := 0
	if ps.conf.SearchMode != "StartsWith" {
		mode = 1
	}
	by := 0
	if ps.conf.SearchBy != "Name" {
		by = 1
	}

	// handle text/drop-down field changes
	sc := func(txt string) {
		ps.settingsChanged = true
	}

	// input fields
	ps.settings.AddCheckbox("Disable AUR: ", disableAur, func(checked bool) {
		ps.settingsChanged = true
		ps.drawSettingsFields(checked, disableCache, separateAurCommands)
		ps.app.SetFocus(ps.settings)
	})
	if !disableAur {
		ps.settings.AddInputField("AUR RPC URL: ", ps.conf.AurRpcUrl, 40, nil, sc).
			AddInputField("AUR timeout (ms): ", strconv.Itoa(ps.conf.AurTimeout), 6, nil, sc).
			AddInputField("AUR search delay (ms): ", strconv.Itoa(ps.conf.AurSearchDelay), 6, nil, sc)
	}
	ps.settings.AddCheckbox("Disable Cache: ", disableCache, func(checked bool) {
		ps.settingsChanged = true
		i, _ := ps.settings.GetFocusedItemIndex()
		ps.drawSettingsFields(disableAur, checked, separateAurCommands)
		ps.settings.SetFocus(i)
		ps.app.SetFocus(ps.settings)
	})
	if !disableCache {
		ps.settings.AddInputField("Cache expiry (m): ", strconv.Itoa(ps.conf.CacheExpiry), 6, nil, sc)
	}
	ps.settings.AddInputField("Max search results: ", strconv.Itoa(ps.conf.MaxResults), 6, nil, sc).
		AddDropDown("Search mode: ", []string{"StartsWith", "Contains"}, mode, nil)
	if dd, ok := ps.settings.GetFormItemByLabel("Search mode: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			if text != ps.conf.SearchMode {
				ps.settingsChanged = true
			}
		})
	}
	ps.settings.AddDropDown("Search by: ", []string{"Name", "Name & Description"}, by, nil)
	if dd, ok := ps.settings.GetFormItemByLabel("Search by: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			if text != ps.conf.SearchBy {
				ps.settingsChanged = true
			}
		})
	}
	ps.settings.AddInputField("Pacman DB path: ", ps.conf.PacmanDbPath, 40, nil, sc).
		AddInputField("Pacman config path: ", ps.conf.PacmanConfigPath, 40, nil, sc).
		AddCheckbox("Separate AUR commands: ", separateAurCommands, func(checked bool) {
			ps.settingsChanged = true
			i, _ := ps.settings.GetFocusedItemIndex()
			ps.drawSettingsFields(disableAur, disableCache, checked)
			ps.settings.SetFocus(i)
			ps.app.SetFocus(ps.settings)
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
		ps.settings.AddInputField("AUR Install command: ", icom, 40, nil, sc).
			AddInputField("AUR Upgrade command: ", ucom, 40, nil, sc)
	}
	ps.settings.AddInputField("Install command: ", ps.conf.InstallCommand, 40, nil, sc).
		AddInputField("Upgrade command: ", ps.conf.SysUpgradeCommand, 40, nil, sc).
		AddInputField("Uninstall command: ", ps.conf.UninstallCommand, 40, nil, sc)

	// key bindings
	ps.settings.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// CTRL + Left navigates to the previous control
		if event.Key() == tcell.KeyLeft && event.Modifiers() == tcell.ModCtrl {
			if ps.prevControl != nil {
				ps.app.SetFocus(ps.prevControl)
			} else {
				ps.app.SetFocus(ps.packages)
			}
			return nil
		}
		// Down / Up / TAB for form navigation
		if event.Key() == tcell.KeyDown ||
			event.Key() == tcell.KeyUp ||
			event.Key() == tcell.KeyTab {
			i, b := ps.settings.GetFocusedItemIndex()
			if b > -1 {
				i = ps.settings.GetFormItemCount() + b
			}
			n := i
			if event.Key() == tcell.KeyUp {
				n-- // move up
			} else {
				n++ // move down
			}
			if i >= 0 && i < ps.settings.GetFormItemCount() {
				// drop downs are excluded from Up / Down handling
				if _, ok := ps.settings.GetFormItem(i).(*tview.DropDown); ok {
					if event.Key() != tcell.KeyTAB && event.Modifiers() != tcell.ModCtrl {
						return event
					}
				}
			}
			// Leave settings from
			if b == ps.settings.GetButtonCount()-1 && event.Key() != tcell.KeyUp {
				ps.app.SetFocus(ps.search)
				return nil
			}
			if i == 0 && event.Key() == tcell.KeyUp {
				ps.app.SetFocus(ps.packages)
				return nil
			}
			ps.settings.SetFocus(n)
			ps.app.SetFocus(ps.settings)
			return nil
		}
		return event
	})
}

// draw package information on screen
func (ps *UI) drawPackageInfo(i InfoRecord, width int) {
	ps.details.Clear()
	ps.details.SetTitle(" "+colorTitle+"[::b]"+i.Name+" ").SetBorderPadding(1, 1, 1, 1)
	r := 0
	ln := 0

	fields, order := getDetailFields(i)
	for _, k := range order {
		//_, _, w, _ := ps.details.GetInnerRect()
		if v, ok := fields[k]; ok {
			if ln == 1 || k == "Last modified" {
				r++
			}
			// split lines if they do not fit on the screen
			w := width - (int(float64(width)*(float64(ps.leftProportion)/10)) + 21) // left box = 40% size, then we use 13 characters for column 0, 2 for padding and 6 for borders
			lines := tview.WordWrap(v, w)
			mr := r
			ps.details.SetCellSimple(r, 0, colorHighlight+"[::b]"+k)
			for _, l := range lines {
				if mr != r {
					ps.details.SetCellSimple(r, 0, "") // we need to add some blank content otherwise it looks weird with some terminal configs
				}
				ps.details.SetCellSimple(r, 1, l)
				r++
			}
			ln++
		}
	}
	ps.details.ScrollToBeginning()
}

// draw packages on screen
func (ps *UI) drawPackages(packages []Package) {
	ps.packages.Clear()

	// header
	ps.drawPackagesHeader()

	// rows
	for i, pkg := range packages {
		color := tcell.ColorGreen
		installed := "-"
		if pkg.IsInstalled {
			installed = "Y"
		}
		if pkg.Source == "AUR" {
			color = tcell.NewRGBColor(23, 147, 209)
		}

		ps.packages.SetCellSimple(i+1, 0, pkg.Name)
		ps.packages.SetCell(i+1, 1, &tview.TableCell{
			Text:            pkg.Source,
			Color:           color,
			BackgroundColor: tcell.ColorBlack,
		})
		ps.packages.SetCell(i+1, 2, &tview.TableCell{
			Text:            installed,
			Expansion:       1000,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorBlack,
		})
	}
	ps.packages.ScrollToBeginning()
}

// adds header row to package table
func (ps *UI) drawPackagesHeader() {
	columns := []string{"Package", "Source", "Installed"}
	for i, col := range columns {
		ps.packages.SetCell(0, i, &tview.TableCell{
			Text:            col,
			NotSelectable:   true,
			Color:           tcell.ColorYellow,
			BackgroundColor: tcell.ColorBlack,
		})
	}
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
		"URL",
		"Votes",
		"Popularity",
		"Last modified",
		"Package URL",
	}

	fields := map[string]string{}
	fields[order[0]] = i.Description
	fields[order[1]] = i.Version
	fields[order[2]] = strings.Join(i.Provides, ", ")
	fields[order[3]] = strings.Join(i.Conflicts, ", ")
	fields[order[4]] = strings.Join(i.License, ", ")
	fields[order[5]] = i.Maintainer

	fields[order[6]] = getDependenciesJoined(i)
	fields[order[7]] = i.URL
	if i.Source == "AUR" {
		fields[order[8]] = fmt.Sprintf("%d", i.NumVotes)
		fields[order[9]] = fmt.Sprintf("%f", i.Popularity)
		fields[order[11]] = aurPackageUrl + i.Name
	} else if util.StringSliceContains(archRepos, i.Source) {
		fields[order[11]] = packageUrl + i.Source + "/" + i.Architecture + "/" + i.Name
	}
	fields[order[10]] = time.Unix(int64(i.LastModified), 0).UTC().Format("2006-01-02 - 15:04:05 (UTC)")

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
	sterm := ps.search.GetText()
	cpkg, exp, found := ps.searchCache.GetWithExpiration(sterm)
	scpkg := cpkg.([]Package)
	if found {
		for i := 0; i < len(scpkg); i++ {
			scpkg[i].IsInstalled = isInstalled(ps.alpmHandle, scpkg[i].Name)
		}
		ps.searchCache.Set(sterm, scpkg, exp.Sub(time.Now()))
	}

	// update currently shown packages
	for i := 1; i < ps.packages.GetRowCount(); i++ {
		newCell := &tview.TableCell{
			Text:            "-",
			Expansion:       1000,
			Color:           tcell.ColorWhite,
			BackgroundColor: tcell.ColorBlack,
		}
		if isInstalled(ps.alpmHandle, ps.packages.GetCell(i, 0).Text) {
			newCell.Text = "Y"
		}
		ps.packages.SetCell(i, 2, newCell)
	}
}
