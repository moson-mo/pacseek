package pacseek

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/rivo/tview"
)

// sets up all our ui components
func (ps *UI) createComponents() {
	// flex grids
	ps.flexRoot = tview.NewFlex().SetDirection(tview.FlexRow)
	ps.flexContainer = tview.NewFlex()
	ps.flexLeft = tview.NewFlex().SetDirection(tview.FlexRow)
	ps.flexTopLeft = tview.NewFlex()
	ps.flexRight = tview.NewFlex().SetDirection(tview.FlexRow)

	// components
	ps.inputSearch = tview.NewInputField()
	ps.tablePackages = tview.NewTable()
	ps.tableDetails = tview.NewTable()
	ps.spinner = tview.NewTextView()
	ps.formSettings = tview.NewForm()
	ps.textMessage = tview.NewTextView()
	ps.textPkgbuild = tview.NewTextView()
	ps.tableNews = tview.NewTable()

	// component config
	ps.flexRoot.SetBorder(true).
		SetTitleAlign(tview.AlignLeft)
	ps.inputSearch.SetLabelStyle(tcell.StyleDefault.Bold(true)).
		SetBorder(true)
	if ps.conf.EnableAutoSuggest {
		ps.inputSearch.SetAutocompleteFunc(ps.autoComplete)
	}
	ps.tableDetails.SetFocusFunc(func() {
		if ps.flexRight.GetItem(0) == ps.textPkgbuild {
			ps.app.SetFocus(ps.textPkgbuild)
		} else {
			ps.app.SetFocus(ps.tablePackages)
		}
	}).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 1, 1)
	ps.displayHelp()
	ps.tablePackages.SetSelectable(true, false).
		SetFixed(1, 1).
		SetBorder(true).
		SetTitleAlign(tview.AlignRight)
	ps.spinner.SetText("").
		SetBorder(true)
	ps.formSettings.
		SetItemPadding(0).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)
	ps.textMessage.SetDynamicColors(true).
		SetBorder(true)
	ps.textPkgbuild.SetWrap(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 1, 1)
	ps.pkgbuildWriter = tview.ANSIWriter(ps.textPkgbuild)
	ps.tableNews.SetSelectable(false, false).
		SetFocusFunc(func() {
			ps.app.SetFocus(ps.inputSearch)
		}).
		SetBorderPadding(1, 1, 1, 1).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)

	// layouting
	ps.flexRoot.AddItem(ps.flexContainer, 0, 1, true).
		AddItem(ps.textMessage, 0, 0, false)
	ps.flexContainer.AddItem(ps.flexLeft, 0, ps.leftProportion, true).
		AddItem(ps.flexRight, 0, 10-ps.leftProportion, false)
	ps.flexLeft.AddItem(ps.flexTopLeft, 3, 1, true).
		AddItem(ps.tablePackages, 0, 1, false)
	ps.flexTopLeft.AddItem(ps.inputSearch, 0, 1, true).
		AddItem(ps.spinner, 3, 1, false)
	ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
}

// apply colors from color scheme
func (ps *UI) applyColors() {
	// containers
	ps.flexRoot.SetTitleColor(ps.conf.Colors().Title).SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.formSettings.SetTitleColor(ps.conf.Colors().Title).SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.tableDetails.SetTitleColor(ps.conf.Colors().Title).SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.inputSearch.SetFieldBackgroundColor(ps.conf.Colors().SearchBar).SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.inputSearch.SetAutocompleteStyles(ps.conf.Colors().SettingsDropdownNotSelected, tcell.StyleDefault, tcell.StyleDefault.Reverse(true))
	ps.textPkgbuild.SetTitleColor(ps.conf.Colors().Title).SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.tableNews.SetTitleColor(ps.conf.Colors().Title).SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.tablePackages.SetBackgroundColor(ps.conf.Colors().DefaultBackground)
	ps.tablePackages.SetSelectedStyle(tcell.StyleDefault.Reverse(true))
	ps.spinner.SetBackgroundColor(ps.conf.Colors().DefaultBackground)

	// settings form
	ps.formSettings.SetFieldBackgroundColor(ps.conf.Colors().SettingsFieldBackground).
		SetFieldTextColor(ps.conf.Colors().SettingsFieldText).
		SetButtonBackgroundColor(ps.conf.Colors().SettingsFieldBackground).
		SetButtonTextColor(ps.conf.Colors().SettingsFieldText).
		SetLabelColor(ps.conf.Colors().SettingsFieldLabel)
	ps.applyDropDownColors()

	// package list
	ps.drawPackageListHeader(ps.conf.PackageColumnWidth)
	for i := 1; i < ps.tablePackages.GetRowCount(); i++ {
		// Package
		c := ps.tablePackages.GetCell(i, 0)
		c.SetBackgroundColor(ps.conf.Colors().DefaultBackground)

		// Source
		c = ps.tablePackages.GetCell(i, 1)
		col := ps.conf.Colors().PackagelistSourceRepository
		if c.Text == "AUR" {
			col = ps.conf.Colors().PackagelistSourceAUR
		}
		c.SetTextColor(col)
		c.SetBackgroundColor(ps.conf.Colors().DefaultBackground)

		// Installed
		c = ps.tablePackages.GetCell(i, 2)
		c.SetTextColor(ps.conf.Colors().DefaultBackground)
		c.SetText(ps.getInstalledStateText(c.Reference.(bool)))
	}

	// details
	if ps.selectedPackage != nil {
		ps.drawPackageInfo(*ps.selectedPackage, ps.width)
	}
}

// apply drop-down colors
func (ps *UI) applyDropDownColors() {
	for _, title := range []string{"Search mode: ", "Search by: ", "Color scheme: ", "Border style: ", "Glyph style: "} {
		if dd, ok := ps.formSettings.GetFormItemByLabel(title).(*tview.DropDown); ok {
			dd.SetListStyles(tcell.StyleDefault.Background(ps.conf.Colors().SettingsDropdownNotSelected).Foreground(ps.conf.Colors().SettingsFieldText),
				tcell.StyleDefault.Background(ps.conf.Colors().SettingsFieldText).Foreground(ps.conf.Colors().SettingsDropdownNotSelected))
		}
	}
}

// replace border characters for ASCII mode
func (ps *UI) applyASCIIMode() {
	tview.Borders.Horizontal = '-'
	tview.Borders.HorizontalFocus = '-'
	tview.Borders.Vertical = '|'
	tview.Borders.VerticalFocus = '|'
	tview.Borders.BottomLeft = '+'
	tview.Borders.BottomLeftFocus = '+'
	tview.Borders.BottomRight = '+'
	tview.Borders.BottomRightFocus = '+'
	tview.Borders.TopLeft = '+'
	tview.Borders.TopLeftFocus = '+'
	tview.Borders.TopRight = '+'
	tview.Borders.TopRightFocus = '+'

	ps.spinner.SetBorder(false).
		SetBorderPadding(1, 1, 1, 1)

	if !strings.HasPrefix(ps.conf.GlyphStyle, "ASCII") {
		ps.conf.SetGlyphStyle("ASCII")
	}
}

// apply colors from color scheme
func (ps *UI) applyGlyphStyle() {

	// titles
	ps.formSettings.SetTitle(" [::b]" + ps.conf.Glyphs().Settings + "Settings ")
	ps.flexRoot.SetTitle(" [::b]" + ps.conf.Glyphs().Package + "pacseek - v" + version + " ")
	ps.tableNews.SetTitle(" [::b]" + ps.conf.Glyphs().Pkgbuild + "Latest news ")

	// package list
	for i := 1; i < ps.tablePackages.GetRowCount(); i++ {
		c := ps.tablePackages.GetCell(i, 2)
		if ref, ok := c.Reference.(bool); ok {
			c.SetText(ps.getInstalledStateText(ref))
		}
	}
}

// set up handlers for keyboard bindings
func (ps *UI) setupKeyBindings() {
	// resize function is called when resize keys are used
	resize := func() {
		ps.flexContainer.ResizeItem(ps.flexLeft, 0, ps.leftProportion)
		ps.flexContainer.ResizeItem(ps.flexRight, 0, 10-ps.leftProportion)
		if ps.selectedPackage != nil {
			ps.drawPackageInfo(*ps.selectedPackage, ps.width)
		}
	}

	// app / global
	ps.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		settingsVisible := ps.flexRight.GetItem(0) == ps.formSettings
		pkgbuildVisible := ps.flexRight.GetItem(0) == ps.textPkgbuild

		// CTRL+Q / ESC - Quit
		if event.Key() == tcell.KeyCtrlQ ||
			(event.Key() == tcell.KeyEscape && !settingsVisible && !pkgbuildVisible && !ps.conf.EnableAutoSuggest) {
			if !ps.settingsChanged {
				if ps.conf.SaveWindowLayout {
					ps.conf.LeftProportion = ps.leftProportion
					ps.saveSettings(false)
				}
				ps.app.Stop()
			}
			ask := tview.NewModal().
				AddButtons([]string{"Yes", "No"}).
				SetText("It seems you've made changes to the settings.\nDo you want to save them?").
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonIndex == 0 {
						ps.saveSettings(false)
					}
					ps.app.Stop()
				})

			ps.app.SetRoot(ask, true)
		}
		// CTRL+S - Show settings
		if event.Key() == tcell.KeyCtrlS ||
			(event.Key() == tcell.KeyEscape && settingsVisible) {
			if !settingsVisible {
				ps.flexRight.Clear()
				ps.flexRight.AddItem(ps.formSettings, 0, 1, false)
				ps.app.SetFocus(ps.formSettings)
			} else {
				ps.flexRight.Clear()
				ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
				ps.app.SetFocus(ps.inputSearch)
				if event.Key() == tcell.KeyEscape {
					ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands, ps.conf.ShowPkgbuildInternally, ps.conf.DisableNewsFeed)
					ps.settingsChanged = false
				}
			}
			return nil
		}
		// CTRL+N - Show help/instructions
		if event.Key() == tcell.KeyCtrlN {
			ps.displayHelp()
			if ps.flexRight.GetItem(0) == ps.formSettings {
				ps.flexRight.Clear()
				ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
			}
			return nil
		}
		// CTRL+U - Upgrade
		if event.Key() == tcell.KeyCtrlU {
			ps.performUpgrade(false)
			return nil
		}
		// CTRL+A - AUR upgrade
		if event.Key() == tcell.KeyCtrlA {
			ps.performUpgrade(true)
			return nil
		}
		// CTRL+B - Show about
		if event.Key() == tcell.KeyCtrlB {
			ps.displayAbout()
			return nil
		}

		// CTRL+W - Wipe cache
		if event.Key() == tcell.KeyCtrlW {
			ps.cacheSearch.Flush()
			ps.cacheInfo.Flush()
			return nil
		}

		// CTRL+P
		if event.Key() == tcell.KeyCtrlP ||
			event.Key() == tcell.KeyEscape && pkgbuildVisible {
			if ps.selectedPackage != nil {
				if pkgbuildVisible {
					ps.flexRight.Clear()
					ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
					ps.app.SetFocus(ps.tablePackages)
				} else {
					if ps.conf.ShowPkgbuildInternally {
						ps.displayPkgbuild()
					} else {
						ps.runCommand(util.Shell(), "-c", ps.getPkgbuildCommand(ps.selectedPackage.Source, ps.selectedPackage.PackageBase))
					}
				}
			}
		}

		// CTRL+O - Open URL for selected package
		if event.Key() == tcell.KeyCtrlO && ps.selectedPackage != nil {
			exec.Command("xdg-open", ps.selectedPackage.URL).Start()
			return nil
		}

		// CTRL+G - Upgradable packages
		if event.Key() == tcell.KeyCtrlG {
			if pkgbuildVisible || settingsVisible {
				ps.flexRight.Clear()
				ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
			}
			ps.displayUpgradable()
			return nil
		}

		// CTRL+L - Locally installed packages
		if event.Key() == tcell.KeyCtrlL {
			if pkgbuildVisible {
				ps.flexRight.Clear()
				ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
			}
			ps.displayInstalled(false)
			return nil
		}

		// Shift+Left - decrease size of left container
		if event.Key() == tcell.KeyLeft && event.Modifiers() == tcell.ModShift {
			if ps.leftProportion != 1 {
				ps.leftProportion--
				resize()
			}
			return nil
		}

		// Shift+Right - increase size of left container
		if event.Key() == tcell.KeyRight && event.Modifiers() == tcell.ModShift {
			if ps.leftProportion != 9 {
				ps.leftProportion++
				resize()
			}
			return nil
		}
		return event
	})

	// redraw package information when the screen is being resized
	ps.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		w, _ := screen.Size()
		if ps.selectedPackage != nil && w != ps.width {
			ps.drawPackageInfo(*ps.selectedPackage, w)
		}
		ps.width = w
		return false
	})

	// input field
	// ENTER / TAB
	ps.inputSearch.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ps.lastSearchTerm = strings.ToLower(ps.inputSearch.GetText())
			if len(ps.lastSearchTerm) == 0 {
				ps.displayInstalled(false)
				return
			} else if len(ps.lastSearchTerm) < 2 {
				ps.displayMessage("Minimum number of characters is 2", true)
				return
			}
			ps.displayPackages(ps.lastSearchTerm)
		} else if key == tcell.KeyTAB {
			ps.app.SetFocus(ps.tablePackages)
		}
	}).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Down
		if event.Key() == tcell.KeyDown && !ps.conf.EnableAutoSuggest {
			ps.app.SetFocus(ps.tablePackages)
			return nil
		}
		// CTRL+Right
		if event.Key() == tcell.KeyRight &&
			event.Modifiers() == tcell.ModCtrl &&
			ps.flexRight.GetItem(0) == ps.formSettings {
			ps.app.SetFocus(ps.formSettings.GetFormItem(0))
			ps.prevComponent = ps.inputSearch
			return nil
		}
		return event
	})

	// packages
	ps.tablePackages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// TAB / Up
		row, _ := ps.tablePackages.GetSelection()
		if event.Key() == tcell.KeyTAB ||
			(event.Key() == tcell.KeyUp && row <= 1) ||
			(event.Key() == tcell.KeyUp && event.Modifiers() == tcell.ModCtrl) {
			if ps.flexRight.GetItem(0) == ps.formSettings && event.Key() == tcell.KeyTAB {
				ps.app.SetFocus(ps.formSettings.GetFormItem(0))
			} else if ps.flexRight.GetItem(0) == ps.textPkgbuild && event.Key() == tcell.KeyTAB {
				ps.app.SetFocus(ps.textPkgbuild)
			} else {
				ps.app.SetFocus(ps.inputSearch)
			}
			return nil
		}
		// Right
		if event.Key() == tcell.KeyRight && ps.flexRight.GetItem(0) == ps.formSettings {
			ps.app.SetFocus(ps.formSettings.GetFormItem(0))
			ps.prevComponent = ps.tablePackages
			return nil
		}
		if event.Key() == tcell.KeyRight && ps.flexRight.GetItem(0) == ps.textPkgbuild {
			ps.app.SetFocus(ps.textPkgbuild)
			ps.prevComponent = ps.tablePackages
			return nil
		}
		// ENTER
		if event.Key() == tcell.KeyEnter {
			ps.installSelectedPackage()
			return nil
		}

		// sorting keys
		if util.SliceContains([]rune{'N', 'S', 'I', 'M', 'P'}, event.Rune()) {
			ps.sortAndRedrawPackageList(event.Rune())
			return nil
		}

		return event
	})
	ps.tablePackages.SetSelectionChangedFunc(func(row, column int) {
		if ps.flexRight.GetItem(0) != ps.tableDetails {
			ps.flexRight.Clear()
			ps.flexRight.AddItem(ps.tableDetails, 0, 1, false)
		}
		ps.displayPackageInfo(row, column)
		ps.tablePackages.SetTitle(fmt.Sprintf(" (%d/%d) ", row, ps.tablePackages.GetRowCount()-1))
	})

	// PKGBUILD
	ps.textPkgbuild.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// CTRL+Left
		if event.Key() == tcell.KeyLeft && event.Modifiers() == tcell.ModCtrl {
			ps.app.SetFocus(ps.tablePackages)
			return nil
		}
		// TAB
		if event.Key() == tcell.KeyTAB {
			ps.app.SetFocus(ps.inputSearch)
			return nil
		}

		return event
	})
}

// sets up settings form
func (ps *UI) setupSettingsForm() {
	// Save button clicked
	ps.formSettings.AddButton("Apply & Save", func() {
		ps.saveSettings(false)
		ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands, ps.conf.ShowPkgbuildInternally, ps.conf.DisableNewsFeed)
	})

	// Defaults button clicked
	ps.formSettings.AddButton("Defaults", func() {
		ps.conf = config.Defaults()
		ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands, ps.conf.ShowPkgbuildInternally, ps.conf.DisableNewsFeed)
		ps.saveSettings(true)
	})

	// add our input fields
	ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands, ps.conf.ShowPkgbuildInternally, ps.conf.DisableNewsFeed)
}

// read settings from from and saves to config file
func (ps *UI) saveSettings(defaults bool) {
	var err error
	for i := 0; i < ps.formSettings.GetFormItemCount(); i++ {
		item := ps.formSettings.GetFormItem(i)
		if input, ok := item.(*tview.InputField); ok {
			txt := input.GetText()
			switch input.GetLabel() {
			case "AUR RPC URL: ":
				ps.conf.AurRpcUrl = txt
			case "AUR timeout (ms): ":
				ps.conf.AurTimeout, err = strconv.Atoi(txt)
				if err != nil {
					ps.displayMessage("Can't convert timeout value to int", true)
					return
				}
			case "AUR search delay (ms): ":
				ps.conf.AurSearchDelay, err = strconv.Atoi(txt)
				if err != nil {
					ps.displayMessage("Can't convert delay value to int", true)
					return
				}
			case "Pacman DB path: ":
				ps.conf.PacmanDbPath = txt
			case "Pacman config path: ":
				ps.conf.PacmanConfigPath = txt
			case "Install command: ":
				ps.conf.InstallCommand = txt
			case "Uninstall command: ":
				ps.conf.UninstallCommand = txt
			case "AUR Install command: ":
				ps.conf.AurInstallCommand = txt
			case "Upgrade command: ":
				ps.conf.SysUpgradeCommand = txt
			case "AUR Upgrade command: ":
				ps.conf.AurUpgradeCommand = txt
			case "Max search results: ":
				ps.conf.MaxResults, err = strconv.Atoi(txt)
				if err != nil {
					ps.displayMessage("Can't convert max results value to int", true)
					return
				}
			case "Cache expiry (m): ":
				ps.conf.CacheExpiry, err = strconv.Atoi(txt)
				if err != nil {
					ps.displayMessage("Can't convert cache expiry value to int", true)
					return
				}
			case "Show PKGBUILD command: ":
				ps.conf.ShowPkgbuildCommand = txt
			case "News-feed URL(s): ":
				ps.conf.FeedURLs = txt
			case "News-feed max items: ":
				ps.conf.FeedMaxItems, err = strconv.Atoi(txt)
				if err != nil {
					ps.displayMessage("Can't convert feed max items value to int", true)
					return
				}
			case "Package column width: ":
				ps.conf.PackageColumnWidth, err = strconv.Atoi(txt)
				if err != nil {
					ps.displayMessage("Can't convert package column width value to int", true)
					return
				}
			}
		} else if dd, ok := item.(*tview.DropDown); ok {
			_, opt := dd.GetCurrentOption()
			switch dd.GetLabel() {
			case "Search mode: ":
				ps.conf.SearchMode = opt
			case "Search by: ":
				ps.conf.SearchBy = opt
			case "Color scheme: ":
				ps.conf.ColorScheme = opt
			case "Border style: ":
				ps.conf.BorderStyle = opt
			case "Glyph style: ":
				ps.conf.GlyphStyle = opt
			}
		} else if cb, ok := item.(*tview.Checkbox); ok {
			switch cb.GetLabel() {
			case "Disable AUR: ":
				ps.conf.DisableAur = cb.IsChecked()
			case "Disable Cache: ":
				ps.conf.DisableCache = cb.IsChecked()
			case "Separate AUR commands: ":
				ps.conf.AurUseDifferentCommands = cb.IsChecked()
			case "Show PKGBUILD internally: ":
				ps.conf.ShowPkgbuildInternally = cb.IsChecked()
			case "Compute \"Required by\": ":
				ps.conf.ComputeRequiredBy = cb.IsChecked()
			case "Disable news-feed: ":
				ps.conf.DisableNewsFeed = cb.IsChecked()
			case "Save window layout: ":
				ps.conf.SaveWindowLayout = cb.IsChecked()
			case "Transparent: ":
				ps.conf.Transparent = cb.IsChecked()
			case "Enable Auto-suggest: ":
				ps.conf.EnableAutoSuggest = cb.IsChecked()
				if ps.conf.EnableAutoSuggest {
					ps.inputSearch.SetAutocompleteFunc(ps.autoComplete)
				} else {
					ps.inputSearch.SetAutocompleteFunc(nil)
				}
			case "Separate Deps with Newline: ":
				ps.conf.SepDepsWithNewLine = cb.IsChecked()
			}
		}
	}
	err = ps.conf.Save()
	if err != nil {
		ps.displayMessage(err.Error(), true)
		return
	}
	msg := "Settings have been applied / saved"
	if defaults {
		msg = "Default settings have been restored"
	}
	ps.displayMessage(msg, false)
	ps.settingsChanged = false
	ps.cacheSearch.Flush()
	if ps.conf.DisableCache {
		ps.cacheInfo.Flush()
	}
}
