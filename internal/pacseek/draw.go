package pacseek

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/gdamore/tcell/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/rivo/tview"
)

// draws input fields on settings form
func (ps *UI) drawSettingsFields(disableAur, disableCache, separateAurCommands, pkgbuildInternal, disableFeed bool) {
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
	gIndex := util.IndexOf(config.GlyphStyles(), ps.conf.GlyphStyle)

	// handle text/drop-down field changes
	sc := func(txt string) {
		ps.settingsChanged = true
	}

	// input fields
	ps.formSettings.AddDropDown("Color scheme: ", config.ColorSchemes(), cIndex, nil)
	if dd, ok := ps.formSettings.GetFormItemByLabel("Color scheme: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			ps.conf.SetColorScheme(text)
			if cb, ok := ps.formSettings.GetFormItemByLabel("Transparent: ").(*tview.Checkbox); ok {
				ps.conf.SetTransparency(cb.IsChecked())
			}
			ps.applyColors()
			if text != ps.conf.ColorScheme {
				ps.settingsChanged = true
			}
		})
	}
	ps.formSettings.AddCheckbox("Transparent: ", ps.conf.Transparent, func(checked bool) {
		ps.settingsChanged = true
		ps.conf.SetTransparency(checked)
		ps.applyColors()
	})
	ps.formSettings.AddDropDown("Border style: ", config.BorderStyles(), bIndex, nil)
	if dd, ok := ps.formSettings.GetFormItemByLabel("Border style: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			ps.conf.SetBorderStyle(text)
			if text != ps.conf.BorderStyle {
				ps.settingsChanged = true
			}
		})
	}
	ps.formSettings.AddDropDown("Glyph style: ", config.GlyphStyles(), gIndex, nil)
	if dd, ok := ps.formSettings.GetFormItemByLabel("Glyph style: ").(*tview.DropDown); ok {
		dd.SetSelectedFunc(func(text string, index int) {
			ps.conf.SetGlyphStyle(text)
			ps.applyGlyphStyle()
			if text != ps.conf.GlyphStyle {
				ps.settingsChanged = true
			}
		})
	}
	ps.formSettings.AddCheckbox("Save window layout: ", ps.conf.SaveWindowLayout, func(checked bool) {
		ps.settingsChanged = true
	})
	ps.formSettings.AddCheckbox("Disable AUR: ", disableAur, func(checked bool) {
		ps.settingsChanged = true
		ps.drawSettingsFields(checked, disableCache, separateAurCommands, pkgbuildInternal, disableFeed)
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
		ps.drawSettingsFields(disableAur, checked, separateAurCommands, pkgbuildInternal, disableFeed)
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
		AddCheckbox("Enable Auto-suggest: ", ps.conf.EnableAutoSuggest, func(checked bool) {
			ps.settingsChanged = true
		}).
		AddCheckbox("Compute \"Required by\": ", ps.conf.ComputeRequiredBy, func(checked bool) {
			ps.settingsChanged = true
		}).
		AddInputField("Pacman DB path: ", ps.conf.PacmanDbPath, 40, nil, sc).
		AddInputField("Pacman config path: ", ps.conf.PacmanConfigPath, 40, nil, sc).
		AddCheckbox("Separate AUR commands: ", separateAurCommands, func(checked bool) {
			ps.settingsChanged = true
			i, _ := ps.formSettings.GetFocusedItemIndex()
			ps.drawSettingsFields(disableAur, disableCache, checked, pkgbuildInternal, disableFeed)
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
		AddInputField("Uninstall command: ", ps.conf.UninstallCommand, 40, nil, sc).
		AddCheckbox("Show PKGBUILD internally: ", pkgbuildInternal, func(checked bool) {
			ps.settingsChanged = true
			i, _ := ps.formSettings.GetFocusedItemIndex()
			ps.drawSettingsFields(disableAur, disableCache, separateAurCommands, checked, disableFeed)
			ps.formSettings.SetFocus(i)
			ps.app.SetFocus(ps.formSettings)
		})
	if !pkgbuildInternal {
		ps.formSettings.AddInputField("Show PKGBUILD command: ", ps.conf.ShowPkgbuildCommand, 40, nil, sc)
	}

	ps.formSettings.AddCheckbox("Disable news-feed: ", disableFeed, func(checked bool) {
		ps.settingsChanged = true
		i, _ := ps.formSettings.GetFocusedItemIndex()
		ps.drawSettingsFields(disableAur, disableCache, separateAurCommands, pkgbuildInternal, checked)
		ps.formSettings.SetFocus(i)
		ps.app.SetFocus(ps.formSettings)
	})
	if !disableFeed {
		ps.formSettings.AddInputField("News-feed URL(s): ", ps.conf.FeedURLs, 40, nil, sc)
		ps.formSettings.AddInputField("News-feed max items: ", strconv.Itoa(ps.conf.FeedMaxItems), 6, nil, sc)
	}

	ps.formSettings.AddInputField("Package column width: ", strconv.Itoa(ps.conf.PackageColumnWidth), 6, nil, func(text string) {
		ps.settingsChanged = true
		width, _ := strconv.Atoi(text)
		ps.drawPackageListContent(ps.shownPackages, width)
	})
	ps.formSettings.AddCheckbox("Separate Deps with Newline: ", ps.conf.SepDepsWithNewLine, func(checked bool) {
		ps.settingsChanged = true
	})

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
	// remove "Latest news" if they were shown previously
	if ps.flexRight.GetItemCount() == 2 {
		ps.flexRight.RemoveItem(ps.flexRight.GetItem(1))
	}

	// clear content and set name
	ps.tableDetails.Clear().
		SetTitle(" [::b]" + ps.conf.Glyphs().Package + i.Name + " ")
	r := 0
	ln := 0

	fields, order := ps.getDetailFields(i)
	maxLen := util.MaxLenMapKey(fields)
	for _, k := range order {
		if v, ok := fields[k]; ok && v != "" {
			if ln == 1 {
				r++
			}
			// split lines if they do not fit on the screen
			w := width - (int(float64(width)*(float64(ps.leftProportion)/10)) + maxLen + 7) // subtract left box, borders, padding and first column
			lines := tview.WordWrap(v, w)
			mr := r
			cell := &tview.TableCell{
				Text:            "[::b]" + k,
				Color:           ps.conf.Colors().Accent,
				BackgroundColor: ps.conf.Colors().DefaultBackground,
			}

			if k == " Show PKGBUILD" {
				ps.tableDetails.SetCellSimple(r, 0, "")
				r++
				cell.SetBackgroundColor(ps.conf.Colors().SearchBar).
					SetTextColor(ps.conf.Colors().SettingsFieldText).
					SetAlign(tview.AlignCenter).
					SetClickedFunc(func() bool {
						if ps.conf.ShowPkgbuildInternally {
							ps.displayPkgbuild()
						} else {
							ps.runCommand(util.Shell(), "-c", v)
						}
						return true
					})
				mr = r
				if i.Source != "AUR" && !util.SliceContains(getArchRepos(), i.Source) {
					lines = tview.WordWrap("-Unofficial repo. PKGBUILD not found. Falling back to the AUR version-", w)
				} else {
					lines = []string{}
				}
			}
			ps.tableDetails.SetCell(r, 0, cell)

			for _, l := range lines {
				if mr != r {
					// we need to add some blank content otherwise it looks weird with some terminal configs
					ps.tableDetails.SetCellSimple(r, 0, "")
				}
				cell := &tview.TableCell{
					Text:            l,
					Color:           tcell.ColorWhite,
					BackgroundColor: ps.conf.Colors().DefaultBackground,
				}
				if k == "Description" {
					cell.SetText("[::b]" + l)
				}
				if k == " Show PKGBUILD" {
					cell.SetText(" " + l).
						SetTextColor(ps.conf.Colors().PackagelistHeader)
				}
				if strings.Contains(k, "URL") {
					cell.SetClickedFunc(func() bool {
						exec.Command("xdg-open", v).Start()
						return true
					})
				}
				if k == "Maintainer" && i.Source == "AUR" {
					cell.SetClickedFunc(func() bool {
						exec.Command("xdg-open", fmt.Sprintf(UrlAurMaintainer, v)).Start()
						return true
					})
				}
				ps.tableDetails.SetCell(r, 1, cell)
				r++
			}
			ln++
			if k == "Package URL" {
				r++
			}
		}
	}
	// check if we got more lines than current screen height
	_, _, _, height := ps.tableDetails.GetInnerRect()
	ps.tableDetailsMore = false
	if r > height-1 {
		ps.tableDetailsMore = true
	}
	ps.tableDetails.ScrollToBeginning()
}

// draw list of upgradable packages
func (ps *UI) drawUpgradable(up []InfoRecord, cached bool) {
	ps.tableDetails.Clear().
		SetTitle(" [::b]" + ps.conf.Glyphs().Upgrades + "Upgradable packages ")

	// draw news if enabled
	if !ps.conf.DisableNewsFeed && ps.flexRight.GetItemCount() != 2 {
		ps.flexRight.AddItem(ps.tableNews, ps.conf.FeedMaxItems+4, 0, false)
		ps.drawNews()
	}

	// header
	columns := []string{"Package  ", "Source  ", "New version  ", "Installed version", ""}
	for i, col := range columns {
		hcell := &tview.TableCell{
			Text:            col,
			Color:           ps.conf.Colors().PackagelistHeader,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
		}
		ps.tableDetails.SetCell(0, i, hcell)
	}

	// lines (not ignored)
	r := 1
	for i := 0; i < len(up); i++ {
		if !up[i].IsIgnored {
			r++
			ps.drawUpgradeableLine(up[i], r, false)
		}
	}
	// lines (ignored)
	for i := 0; i < len(up); i++ {
		if up[i].IsIgnored {
			r++
			ps.drawUpgradeableLine(up[i], r, true)
		}
	}

	// no updates found message else sysupgrade button
	r += 2
	if len(up) == 0 {
		ps.tableDetails.SetCell(r, 0, &tview.TableCell{
			Text:            "No upgrades found",
			Color:           ps.conf.Colors().PackagelistHeader,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
		})
	} else {
		ps.tableDetails.SetCell(r, 0, &tview.TableCell{
			Text:            " [::b]Sysupgrade",
			Align:           tview.AlignCenter,
			Color:           ps.conf.Colors().SettingsFieldText,
			BackgroundColor: ps.conf.Colors().SearchBar,
			Clicked: func() bool {
				ps.performUpgrade(false)
				ps.cacheInfo.Delete("#upgrades#")
				ps.displayUpgradable()
				return true
			},
		})
	}

	// refresh button
	if cached {
		r += 2
		ps.tableDetails.SetCell(r, 0, &tview.TableCell{
			Text:            " [::b]Refresh",
			Color:           ps.conf.Colors().SettingsFieldText,
			BackgroundColor: ps.conf.Colors().SearchBar,
			Align:           tview.AlignCenter,
			Clicked: func() bool {
				ps.cacheInfo.Delete("#upgrades#")
				ps.displayUpgradable()
				return true
			},
		})
	}

	// set nil to avoid printing package details when resizing
	// somewhat hacky, needs refactoring (well, everything needs refactoring here)
	ps.selectedPackage = nil
}

// draw news items
func (ps *UI) drawNews() {
	go func() {
		news, err := getNews(ps.conf.FeedURLs, ps.conf.FeedMaxItems)
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.tableNews.SetCellSimple(0, 0, "Failed fetching feed(s): "+err.Error())
			})
			return
		}

		ps.app.QueueUpdateDraw(func() {
			for r, item := range news {
				item := item
				ps.tableNews.SetCell(r, 0, &tview.TableCell{
					Text: "* [::u]" + item.Title,
					Clicked: func() bool {
						exec.Command("xdg-open", item.Link).Start()
						return true
					},
					Color:           tcell.ColorWhite,
					BackgroundColor: ps.conf.Colors().DefaultBackground,
				}).
					SetCellSimple(r, 1, "("+item.PublishedParsed.Format("2006-01-02")+")")
			}
		})
	}()
}

// draws a line for an upgradable package
func (ps *UI) drawUpgradeableLine(up InfoRecord, lNum int, ignored bool) {
	cellDesc := &tview.TableCell{
		Text:            "[::b]" + up.Name,
		Color:           ps.conf.Colors().Accent,
		BackgroundColor: ps.conf.Colors().DefaultBackground,
		Clicked: func() bool {
			ps.selectedPackage = &up
			ps.drawPackageInfo(up, ps.width)
			return true
		},
	}
	cellSource := &tview.TableCell{
		Text:            up.Source,
		Color:           ps.conf.Colors().Accent,
		BackgroundColor: ps.conf.Colors().DefaultBackground,
	}
	cellVnew := &tview.TableCell{
		Text:            "[::b]" + up.Version,
		Color:           ps.conf.Colors().PackagelistSourceRepository,
		BackgroundColor: ps.conf.Colors().DefaultBackground,
		Clicked: func() bool {
			if ps.conf.ShowPkgbuildInternally {
				ps.selectedPackage = &up
				ps.displayPkgbuild()
			} else {
				ps.runCommand(util.Shell(), "-c", ps.getPkgbuildCommand(up.Source, up.PackageBase))
			}
			return true
		},
	}
	cellVold := &tview.TableCell{
		Text:            up.LocalVersion,
		Color:           ps.conf.Colors().PackagelistSourceAUR,
		BackgroundColor: ps.conf.Colors().DefaultBackground,
	}

	ps.tableDetails.SetCell(lNum, 0, cellDesc).
		SetCell(lNum, 1, cellSource).
		SetCell(lNum, 2, cellVnew).
		SetCell(lNum, 3, cellVold)

	// rebuild button for AUR packages
	if up.Source == "AUR" && !ignored {
		cellRebuild := &tview.TableCell{
			Text:            " [::b]Rebuild / Update",
			Color:           ps.conf.Colors().SettingsFieldText,
			BackgroundColor: ps.conf.Colors().SearchBar,
			Clicked: func() bool {
				// sorry not very optimal
				var pkglist []PkgStatus = nil
				pkglist = append(pkglist, PkgStatus{up, false})
				ps.installSelectedPackages(pkglist)
				ps.cacheInfo.Delete("#upgrades#")
				ps.displayUpgradable()
				return true
			},
		}
		ps.tableDetails.SetCell(lNum, 4, cellRebuild)
	}

	if ignored {
		cellDesc.SetTextColor(ps.conf.Colors().PackagelistHeader)
		cellVnew.SetTextColor(ps.conf.Colors().PackagelistHeader)
		cellIgnored := &tview.TableCell{
			Text:            "ignored",
			Color:           ps.conf.Colors().PackagelistHeader,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
		}
		ps.tableDetails.SetCell(lNum, 4, cellIgnored)
	}
}

// draw packages on screen
func (ps *UI) drawPackageListContent(packages []Package, pkgwidth int) {
	ps.tablePackages.Clear()

	// header
	ps.drawPackageListHeader(pkgwidth)

	// rows
	for i, pkg := range packages {
		color := ps.conf.Colors().PackagelistSourceRepository

		if pkg.Source == "AUR" {
			color = ps.conf.Colors().PackagelistSourceAUR
		}

		// necessary conversion for boolcast workaround
		var isInstalled int8
		if pkg.IsInstalled {
			isInstalled = 0x1
		} else {
			isInstalled = 0x0
		}

		ps.tablePackages.SetCell(i+1, 0, &tview.TableCell{
			Text:            pkg.Name,
			Color:           tcell.ColorWhite,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
			MaxWidth:        pkgwidth,
		}).
			SetCell(i+1, 1, &tview.TableCell{
				Text:            pkg.Source,
				Color:           color,
				BackgroundColor: ps.conf.Colors().DefaultBackground,
			}).
			SetCell(i+1, 2, &tview.TableCell{
				Color:       ps.conf.Colors().DefaultBackground,
				Text:        ps.getInstalledStateText(isInstalled | pkg.IsMarked),
				Expansion:   1000,
				Reference:   isInstalled | pkg.IsMarked,
				Transparent: true,
			})
	}
	ps.tablePackages.ScrollToBeginning()
}

// draw pkgbuild on screen
func (ps *UI) drawPkgbuild(content, pkg string) {
	ps.textPkgbuild.SetTitle(" [::b]" + ps.conf.Glyphs().Pkgbuild + "PKGBUILD - " + pkg + " ")
	err := quick.Highlight(ps.pkgbuildWriter, tview.Escape(content), "bash", "terminal16m", ps.conf.Colors().StylePKGBUILD)
	if err != nil {
		ps.textPkgbuild.SetText(err.Error())
		return
	}
	ps.textPkgbuild.ScrollToBeginning()
}

// adds header row to package table
func (ps *UI) drawPackageListHeader(pkgwidth int) {
	columns := []string{"Package", "Source", "Installed"}
	for i, col := range columns {
		col := col
		width := 0
		if i == 0 {
			width = pkgwidth
			col = fmt.Sprintf("%-"+strconv.Itoa(width)+"s", col)
		}
		ps.tablePackages.SetCell(0, i, &tview.TableCell{
			Text:            col,
			NotSelectable:   true,
			Color:           ps.conf.Colors().PackagelistHeader,
			BackgroundColor: ps.conf.Colors().DefaultBackground,
			MaxWidth:        width,
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
	ps.drawPackageListContent(ps.shownPackages, ps.conf.PackageColumnWidth)
	ps.tablePackages.Select(1, 0)
}

// composes a map with fields and values (package information) for our details box
func (ps *UI) getDetailFields(i InfoRecord) (map[string]string, []string) {
	order := []string{
		"Description",
		"Version",
		"Maintainer",
		"Licenses",
		"Votes",
		"Popularity",
		"Last modified",
		"Flagged out of date",
		"URL",
		"Package URL",
		"Provides",
		"Conflicts",
		"Required by",
		"Dependencies",
		" Show PKGBUILD", //the space in front is an ugly alignment hack ;)
	}

	fields := map[string]string{}
	fields["Description"] = i.Description
	fields["Version"] = i.Version
	fields["Provides"] = strings.Join(i.Provides, ", ")
	fields["Conflicts"] = strings.Join(i.Conflicts, ", ")
	fields["Licenses"] = strings.Join(i.License, ", ")
	fields["Maintainer"] = i.Maintainer
	fields["Dependencies"] = getDependenciesJoined(i, ps.getInstalledStateText(0x1), ps.getInstalledStateText(0x0), ps.conf.SepDepsWithNewLine)
	fields["Required by"] = strings.Join(i.RequiredBy, ", ")
	fields["URL"] = i.URL
	if i.Source == "AUR" {
		fields["Votes"] = fmt.Sprintf("%d", i.NumVotes)
		fields["Popularity"] = fmt.Sprintf("%f", i.Popularity)
		fields["Package URL"] = fmt.Sprintf(UrlAurPackage, i.Name)
	} else if (!ps.isArm && util.SliceContains(getArchRepos(), i.Source)) ||
		ps.isArm && util.SliceContains(getArchArmRepos(), i.Source) {
		if ps.isArm {
			fields["Package URL"] = fmt.Sprintf(UrlArmPackage, i.Architecture, i.Name)
		} else {
			fields["Package URL"] = fmt.Sprintf(UrlPackage, i.Source, i.Architecture, i.Name)
		}
	}
	if i.LastModified != 0 {
		fields["Last modified"] = time.Unix(int64(i.LastModified), 0).UTC().Format("2006-01-02 - 15:04:05 (UTC)")
	}
	if i.OutOfDate != 0 {
		fields["Flagged out of date"] = time.Unix(int64(i.OutOfDate), 0).UTC().Format("[red]2006-01-02 - 15:04:05 (UTC)")
	}
	if !ps.isArm || (ps.isArm && i.Source == "AUR") {
		fields[" Show PKGBUILD"] = ps.getPkgbuildCommand(i.Source, i.PackageBase)
	}

	return fields, order
}

// join and format different dependencies as string
func getDependenciesJoined(i InfoRecord, installedIcon, notInstalledicon string, newline bool) string {
	deps := []string{}
	for _, dep := range i.DepsAndSatisfiers {
		add := ""
		if dep.Installed {
			add += installedIcon
		} else {
			add += notInstalledicon
		}
		add += " " + dep.DepName
		if dep.DepType != "dep" {
			add += " (" + dep.DepType + ")"
		}
		deps = append(deps, add)
	}
	separator := ", "
	if newline {
		separator = "\n"
	}
	return strings.Join(deps, separator)
}

// this function maybe redundant i.e. needs probably rework
// those functions are inefective and probably slow when big list
func (ps *UI) getPkgMarked(pkgname string) int8 {
	for i := range ps.shownPackages {
		if ps.shownPackages[i].Name == pkgname {
			return ps.shownPackages[i].IsMarked
		}
	}

	return 0x0
}

func (ps *UI) setPkgMarked(pkgname string, ismarked int8) {
	for i := range ps.shownPackages {
		if ps.shownPackages[i].Name == pkgname {
			ps.shownPackages[i].IsMarked = ismarked
			break
		}
	}
}

// updates the "install state" of all packages in cache and package list
func (ps *UI) updateInstalledState() {
	// update cached packages
	sterm := strings.ToLower(ps.inputSearch.GetText())
	cpkg, exp, found := ps.cacheSearch.GetWithExpiration(sterm)
	if found {
		scpkg := cpkg.([]Package)
		for i := 0; i < len(scpkg); i++ {
			scpkg[i].IsInstalled = isPackageInstalled(ps.alpmHandle, scpkg[i].Name)
			scpkg[i].IsMarked = ps.getPkgMarked(scpkg[i].Name)
		}
		ps.cacheSearch.Set(sterm, scpkg, time.Until(exp))
	}

	// update currently shown packages
	for i := 1; i < ps.tablePackages.GetRowCount(); i++ {
		var isInstalled int8
		if isPackageInstalled(ps.alpmHandle, ps.tablePackages.GetCell(i, 0).Text) {
			isInstalled = 0x1
		} else {
			isInstalled = 0x0
		}

		isMarked := ps.getPkgMarked(ps.tablePackages.GetCell(i, 0).Text)

		newCell := &tview.TableCell{
			Text:        ps.getInstalledStateText(isInstalled | isMarked),
			Expansion:   1000,
			Reference:   isInstalled | isMarked,
			Transparent: true,
		}
		ps.tablePackages.SetCell(i, 2, newCell)
	}
}

// compose text for "Installed" column in package list
func (ps *UI) getInstalledStateText(state int8) string {
	glyphs := ps.conf.Glyphs()
	colStrInstalled := "[#ff0000::b]"
	installed := glyphs.NotInstalled

	// isInstalled == true
	if state&0x1 == 0x1 {
		installed = glyphs.Installed
		colStrInstalled = "[#00ff00::b]"
	}

	// isMarked == true
	if state&0x2 == 0x2 {
		installed = glyphs.Marked
		if state&0x1 == 0x1 {
			colStrInstalled = "[#ff0000::b]"
		} else {
			colStrInstalled = "[#00ff00::b]"
		}
	}

	if ps.conf.ColorScheme == "Monochrome" || ps.flags.MonochromeMode {
		colStrInstalled = "[white:black:b]"
	}

	whiteBlack := "[white:black:-]"
	if ps.conf.Colors().DefaultBackground == tcell.ColorDefault {
		whiteBlack = "[white:-:-]"
	}
	ret := whiteBlack + glyphs.PrefixState + colStrInstalled + installed + whiteBlack + glyphs.SuffixState

	return ret
}
