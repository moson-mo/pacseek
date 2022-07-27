package pacseek

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/rivo/tview"
)

// sets up all our ui components
func (ps *UI) setupComponents() {
	// flex grids
	ps.root = tview.NewFlex().SetDirection(tview.FlexRow)
	ps.container = tview.NewFlex()
	ps.left = tview.NewFlex().SetDirection(tview.FlexRow)
	ps.topleft = tview.NewFlex()
	ps.right = tview.NewFlex().SetDirection(tview.FlexRow)

	// components
	ps.search = tview.NewInputField()
	ps.packages = tview.NewTable()
	ps.details = tview.NewTable()
	ps.spinner = tview.NewTextView()
	ps.settings = tview.NewForm()
	ps.status = tview.NewTextView()

	// component config
	ps.root.SetBorder(true).
		SetTitle(" [::b]pacseek - v" + version + " ").
		SetTitleAlign(tview.AlignLeft)
	ps.search.SetLabelStyle(tcell.StyleDefault.Bold(true)).
		SetBorder(true)
	ps.details.SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 1, 1)
	ps.showHelp()
	ps.packages.SetSelectable(true, false).
		SetFixed(1, 1).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)
	ps.spinner.SetText("").
		SetBorder(true)
	ps.settings.
		SetItemPadding(0).
		SetBorder(true).
		SetTitle(" [::b]Settings ").
		SetTitleAlign(tview.AlignLeft)
	ps.status.SetDynamicColors(true).
		SetBorder(true)

	// layouting
	ps.leftProportion = 4
	ps.root.AddItem(ps.container, 0, 1, true).
		AddItem(ps.status, 0, 0, false)
	ps.container.AddItem(ps.left, 0, ps.leftProportion, true).
		AddItem(ps.right, 0, 10-ps.leftProportion, false)
	ps.left.AddItem(ps.topleft, 3, 1, true).
		AddItem(ps.packages, 0, 1, false)
	ps.topleft.AddItem(ps.search, 0, 1, true).
		AddItem(ps.spinner, 3, 1, false)
	ps.right.AddItem(ps.details, 0, 1, false)
}

// apply colors from color scheme
func (ps *UI) setupColors() {
	// containers
	ps.root.SetTitleColor(ps.conf.Colors().Title)
	ps.settings.SetTitleColor(ps.conf.Colors().Title)
	ps.details.SetTitleColor(ps.conf.Colors().Title)
	ps.search.SetFieldBackgroundColor(ps.conf.Colors().SearchBar)

	// settings form
	ps.settings.SetFieldBackgroundColor(ps.conf.Colors().SettingsFieldBackground).
		SetFieldTextColor(ps.conf.Colors().SettingsFieldText).
		SetButtonBackgroundColor(ps.conf.Colors().SettingsFieldBackground).
		SetButtonTextColor(ps.conf.Colors().SettingsFieldText).
		SetLabelColor(ps.conf.Colors().SettingsFieldLabel)
	ps.setupDropDownColors()

	// package list
	ps.drawPackagesHeader()
	for i := 1; i < ps.packages.GetRowCount(); i++ {
		c := ps.packages.GetCell(i, 1)
		col := ps.conf.Colors().PackagelistSourceRepository
		if c.Text == "AUR" {
			col = ps.conf.Colors().PackagelistSourceAUR
		}
		c.SetTextColor(col)
	}

	// details
	if ps.details.GetCell(0, 0) != nil && ps.details.GetCell(0, 0).Text == "[::b]Description" {
		for i := 0; i < ps.details.GetRowCount(); i++ {
			ps.details.GetCell(i, 0).SetTextColor(ps.conf.Colors().Accent)
		}
	}
}

// apply drop-down colors
func (ps *UI) setupDropDownColors() {
	for _, title := range []string{"Search mode: ", "Search by: ", "Color scheme: ", "Border style: "} {
		if dd, ok := ps.settings.GetFormItemByLabel(title).(*tview.DropDown); ok {
			dd.SetListStyles(tcell.StyleDefault.Background(ps.conf.Colors().SettingsDropdownNotSelected).Foreground(ps.conf.Colors().SettingsFieldText),
				tcell.StyleDefault.Background(ps.conf.Colors().SettingsFieldText).Foreground(ps.conf.Colors().SettingsDropdownNotSelected))
		}
	}
}

// replace border characters for ASCII mode
func (ps *UI) setASCIIMode() {
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
}

// set up handlers for keyboard bindings
func (ps *UI) setupKeyBindings() {
	// resize function is called when resize keys are used
	resize := func() {
		ps.container.ResizeItem(ps.left, 0, ps.leftProportion)
		ps.container.ResizeItem(ps.right, 0, 10-ps.leftProportion)
		if ps.selectedPackage != nil {
			ps.drawPackageInfo(*ps.selectedPackage, ps.width)
		}
	}

	// app / global
	ps.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		settingsVisible := ps.right.GetItem(0) == ps.settings
		// CTRL+Q - Quit
		if event.Key() == tcell.KeyCtrlQ ||
			(event.Key() == tcell.KeyEscape && !settingsVisible) {
			ps.alpmHandle.Release()
			if !ps.settingsChanged {
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
				ps.right.Clear()
				ps.right.AddItem(ps.settings, 0, 1, false)
			} else {
				ps.right.Clear()
				ps.right.AddItem(ps.details, 0, 1, false)
				ps.app.SetFocus(ps.search)
				if event.Key() == tcell.KeyEscape {
					ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands)
					ps.settingsChanged = false
				}
			}
			return nil
		}
		// CTRL+N - Show help/instructions
		if event.Key() == tcell.KeyCtrlN {
			ps.showHelp()
			if ps.right.GetItem(0) == ps.settings {
				ps.right.Clear()
				ps.right.AddItem(ps.details, 0, 1, false)
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
			ps.showAbout()
			return nil
		}

		// CTRL+W - Wipe cache
		if event.Key() == tcell.KeyCtrlW {
			ps.searchCache.Flush()
			ps.infoCache.Flush()
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

	// search
	ps.search.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// TAB / Down
		if event.Key() == tcell.KeyTAB || event.Key() == tcell.KeyDown {
			ps.app.SetFocus(ps.packages)
			return nil
		}
		// ENTER
		if event.Key() == tcell.KeyEnter {
			ps.lastTerm = ps.search.GetText()
			if len(ps.lastTerm) < 2 {
				ps.showMessage("Minimum number of characters is 2", true)
				return nil
			}
			ps.showPackages(ps.lastTerm)
			return nil
		}
		// CTRL+Right
		if event.Key() == tcell.KeyRight &&
			event.Modifiers() == tcell.ModCtrl &&
			ps.right.GetItem(0) == ps.settings {
			ps.app.SetFocus(ps.settings.GetFormItem(0))
			ps.prevControl = ps.search
			return nil
		}
		return event
	})

	// packages
	ps.packages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// TAB / Up
		row, _ := ps.packages.GetSelection()
		if event.Key() == tcell.KeyTAB ||
			(event.Key() == tcell.KeyUp && row <= 1) ||
			(event.Key() == tcell.KeyUp && event.Modifiers() == tcell.ModCtrl) {
			if ps.right.GetItem(0) == ps.settings && event.Key() == tcell.KeyTAB {
				ps.app.SetFocus(ps.settings.GetFormItem(0))
			} else {
				ps.app.SetFocus(ps.search)
			}
			return nil
		}
		// Right
		if event.Key() == tcell.KeyRight && ps.right.GetItem(0) == ps.settings {
			ps.app.SetFocus(ps.settings.GetFormItem(0))
			ps.prevControl = ps.packages
			return nil
		}
		// ENTER
		if event.Key() == tcell.KeyEnter {
			ps.installPackage()
			return nil
		}

		// sorting keys
		if util.SliceContains([]rune{'N', 'S', 'I', 'M', 'P'}, event.Rune()) {
			ps.sortAndRedrawPkgList(event.Rune())
			return nil
		}

		return event
	})
	ps.packages.SetSelectionChangedFunc(ps.showPackageInfo)
}

// sets up settings form
func (ps *UI) setupSettingsForm() {
	// Save button clicked
	ps.settings.AddButton("Apply & Save", func() {
		ps.saveSettings(false)
		ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands)
	})

	// Defaults button clicked
	ps.settings.AddButton("Defaults", func() {
		ps.conf = config.Defaults()
		ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands)
		ps.saveSettings(true)
	})

	// add our input fields
	ps.drawSettingsFields(ps.conf.DisableAur, ps.conf.DisableCache, ps.conf.AurUseDifferentCommands)
}

// read settings from from and saves to config file
func (ps *UI) saveSettings(defaults bool) {
	var err error
	for i := 0; i < ps.settings.GetFormItemCount(); i++ {
		item := ps.settings.GetFormItem(i)
		if input, ok := item.(*tview.InputField); ok {
			txt := input.GetText()
			switch input.GetLabel() {
			case "AUR RPC URL: ":
				ps.conf.AurRpcUrl = txt
			case "AUR timeout (ms): ":
				ps.conf.AurTimeout, err = strconv.Atoi(txt)
				if err != nil {
					ps.showMessage("Can't convert timeout value to int", true)
					return
				}
			case "AUR search delay (ms): ":
				ps.conf.AurSearchDelay, err = strconv.Atoi(txt)
				if err != nil {
					ps.showMessage("Can't convert delay value to int", true)
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
					ps.showMessage("Can't convert max results value to int", true)
					return
				}
			case "Cache expiry (m): ":
				ps.conf.CacheExpiry, err = strconv.Atoi(txt)
				if err != nil {
					ps.showMessage("Can't convert cache expiry value to int", true)
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
			}
		} else if cb, ok := item.(*tview.Checkbox); ok {
			switch cb.GetLabel() {
			case "Disable AUR: ":
				ps.conf.DisableAur = cb.IsChecked()
			case "Disable Cache: ":
				ps.conf.DisableCache = cb.IsChecked()
			case "Separate AUR commands: ":
				ps.conf.AurUseDifferentCommands = cb.IsChecked()
			}
		}
	}
	err = ps.conf.Save()
	if err != nil {
		ps.showMessage(err.Error(), true)
		return
	}
	msg := "Settings have been applied / saved"
	if defaults {
		msg = "Default settings have been restored"
	}
	ps.showMessage(msg, false)
	ps.settingsChanged = false
	ps.searchCache.Flush()
	if ps.conf.DisableCache {
		ps.infoCache.Flush()
	}
}
