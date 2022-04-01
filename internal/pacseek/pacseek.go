package pacseek

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Jguer/go-alpm/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/rivo/tview"
)

// UI is holding our application information and all tview components
type UI struct {
	conf *config.Settings
	app  *tview.Application

	alpmHandle *alpm.Handle

	root      *tview.Flex
	left      *tview.Flex
	topleft   *tview.Flex
	right     *tview.Flex
	container *tview.Flex

	search      *tview.InputField
	packages    *tview.Table
	details     *tview.Table
	spinner     *tview.TextView
	settings    *tview.Form
	status      *tview.TextView
	prevControl tview.Primitive

	locker        *sync.RWMutex
	messageLocker *sync.RWMutex

	quitSpin        chan bool
	width           int
	selectedPackage *InfoRecord
	settingsChanged bool
}

// New creates a UI object and makes sure everything is initialized
func New(config *config.Settings) (*UI, error) {
	ui := UI{
		conf:            config,
		app:             tview.NewApplication(),
		locker:          &sync.RWMutex{},
		messageLocker:   &sync.RWMutex{},
		quitSpin:        make(chan bool),
		settingsChanged: false,
	}

	var err error
	ui.alpmHandle, err = initPacmanDbs(config.PacmanDbPath, config.PacmanConfigPath)
	if err != nil {
		return nil, err
	}

	// setup UI
	ui.setupComponents()
	ui.setupKeyBindings()
	ui.setupSettingsForm()

	return &ui, nil
}

// Start runs application / event-loop
func (ps *UI) Start(term string) error {
	if term != "" {
		ps.search.SetText(term)
		ps.showPackages(term)
	}
	return ps.app.SetRoot(ps.root, true).EnableMouse(true).Run()
}

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
		SetTitle(" [#00dfff][::b]pacseek - v0.2.5 ").
		SetTitleAlign(tview.AlignLeft)
	ps.search.SetLabelStyle(tcell.StyleDefault.Attributes(tcell.AttrBold)).
		SetFieldBackgroundColor(tcell.NewRGBColor(5, 100, 160)).
		SetBorder(true)
	ps.details.SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 1, 1)
	ps.showHelp()
	ps.packages.SetSelectable(true, false).
		SetFixed(1, 1).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)
	ps.packages.SetCell(0, 0, &tview.TableCell{
		Text:          "Package - Source - Installed",
		NotSelectable: true,
		Color:         tcell.ColorYellow,
	})
	ps.spinner.SetText("|").
		SetBorder(true)
	ps.settings.SetBorder(true).
		SetTitle(" [#00dfff][::b]Settings ").
		SetTitleAlign(tview.AlignLeft)
	ps.status.SetDynamicColors(true).
		SetBorder(true)

	// layouting
	ps.root.AddItem(ps.container, 0, 1, true)
	ps.root.AddItem(ps.status, 0, 0, false)
	ps.container.AddItem(ps.left, 0, 4, true)
	ps.container.AddItem(ps.right, 0, 6, false)
	ps.left.AddItem(ps.topleft, 3, 1, true)
	ps.topleft.AddItem(ps.search, 0, 1, true)
	ps.topleft.AddItem(ps.spinner, 3, 1, false)
	ps.left.AddItem(ps.packages, 0, 1, false)
	ps.right.AddItem(ps.details, 0, 1, false)
}

// set up handlers for keyboard bindings
func (ps *UI) setupKeyBindings() {
	// app / global
	ps.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// CTRL+Q
		if event.Key() == tcell.KeyCtrlQ {
			ps.alpmHandle.Release()
			if ps.settingsChanged {
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
			} else {
				ps.app.Stop()
			}
		}
		// CTRL+S
		if event.Key() == tcell.KeyCtrlS {
			if ps.right.GetItem(0) != ps.settings {
				ps.right.Clear()
				ps.right.AddItem(ps.settings, 0, 1, false)
			} else {
				ps.right.Clear()
				ps.right.AddItem(ps.details, 0, 1, false)
				ps.app.SetFocus(ps.search)
			}
			return nil
		}
		// CTRL+H
		if event.Key() == tcell.KeyCtrlH {
			ps.showHelp()
			if ps.right.GetItem(0) == ps.settings {
				ps.right.Clear()
				ps.right.AddItem(ps.details, 0, 1, false)
			}
			return nil
		}
		// CTRL+U
		if event.Key() == tcell.KeyCtrlU {
			ps.performSyncSysUpgrade()
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
			ps.showPackages(ps.search.GetText())
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
		return event
	})
	ps.packages.SetSelectionChangedFunc(ps.showPackageInfo)
}

// sets up settings form
func (ps *UI) setupSettingsForm() {
	addFields := func() {
		mode := 1
		if ps.conf.SearchMode != "Contains" {
			mode = 0
		}

		sc := func(txt string) {
			ps.settingsChanged = true
		}

		// input fields
		ps.settings.AddInputField("AUR RPC URL: ", ps.conf.AurRpcUrl, 40, nil, sc)
		ps.settings.AddInputField("AUR timeout (ms): ", strconv.Itoa(ps.conf.AurTimeout), 6, nil, sc)
		ps.settings.AddInputField("AUR search delay (ms): ", strconv.Itoa(ps.conf.AurSearchDelay), 6, nil, sc)
		ps.settings.AddInputField("Max search results: ", strconv.Itoa(ps.conf.MaxResults), 6, nil, sc)
		ps.settings.AddInputField("Pacman DB path: ", ps.conf.PacmanDbPath, 40, nil, sc)
		ps.settings.AddInputField("Pacman config path: ", ps.conf.PacmanConfigPath, 40, nil, sc)
		ps.settings.AddInputField("Install command: ", ps.conf.InstallCommand, 40, nil, sc)
		ps.settings.AddInputField("Uninstall command: ", ps.conf.UninstallCommand, 40, nil, sc)
		ps.settings.AddInputField("Upgrade command: ", ps.conf.SysUpgradeCommand, 40, nil, sc)
		ps.settings.AddDropDown("Search mode: ", []string{"StartsWith", "Contains"}, mode, nil)
		if dd, ok := ps.settings.GetFormItemByLabel("Search mode: ").(*tview.DropDown); ok {
			dd.SetSelectedFunc(func(text string, index int) {
				if text != ps.conf.SearchMode {
					ps.settingsChanged = true
				}
			})
		}

		// key bindings
		ps.settings.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyLeft && event.Modifiers() == tcell.ModCtrl {
				if ps.prevControl != nil {
					ps.app.SetFocus(ps.prevControl)
				} else {
					ps.app.SetFocus(ps.packages)
				}
				return nil
			}
			return event
		})

		// form input items
		for i := 0; i < ps.settings.GetFormItemCount(); i++ {
			item := ps.settings.GetFormItem(i)
			if i+1 < ps.settings.GetFormItemCount() {
				next := ps.settings.GetFormItem(i + 1)
				var prev tview.FormItem
				if i > 0 {
					prev = ps.settings.GetFormItem(i - 1)
				}
				item.SetFinishedFunc(func(key tcell.Key) {
					if key == tcell.KeyUp {
						if prev != nil {
							ps.app.SetFocus(prev)
							return
						}
						ps.app.SetFocus(ps.packages)
					} else {
						ps.app.SetFocus(next)
					}
				})
			} else {
				item.SetFinishedFunc(func(key tcell.Key) {
					ps.app.SetFocus(ps.settings.GetButton(0))
				})
			}
		}

		// Save button
		ps.settings.GetButton(0).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTAB || event.Key() == tcell.KeyDown {
				ps.app.SetFocus(ps.settings.GetButton(1))
				return nil
			}
			if event.Key() == tcell.KeyUp {
				ps.app.SetFocus(ps.settings.GetFormItem(ps.settings.GetFormItemCount() - 1))
			}
			return event
		})

		// Defaults button
		ps.settings.GetButton(1).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTAB || event.Key() == tcell.KeyDown {
				ps.app.SetFocus(ps.search)
				return nil
			}
			if event.Key() == tcell.KeyUp {
				ps.app.SetFocus(ps.settings.GetButton(0))
			}
			return event
		})
	}

	// Save button clicked
	ps.settings.AddButton("Apply & Save", func() {
		ps.saveSettings(false)
	})

	// Defaults button clicked
	ps.settings.AddButton("Defaults", func() {
		ps.conf = config.Defaults()
		ps.settings.Clear(false)
		addFields()
		ps.saveSettings(true)
	})

	// add our input fields
	addFields()
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
		if source == "AUR" {
			time.Sleep(time.Duration(ps.conf.AurSearchDelay) * time.Millisecond)
		}

		if !ps.isSelected(pkg, true) {
			return
		}
		ps.app.QueueUpdateDraw(func() {
			ps.details.SetTitle(" [#00dfff][::b]" + pkg + " - Retrieving data... ")
		})

		ps.locker.Lock()
		ps.startSpin()
		defer ps.stopSpin()
		defer ps.locker.Unlock()

		var info RpcResult
		if source == "AUR" {
			info = infoAur(ps.conf.AurRpcUrl, pkg, ps.conf.AurTimeout)
		} else {
			info = infoPacman(ps.alpmHandle, pkg)
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
				ps.details.SetCellSimple(0, 0, fmt.Sprintf("[red]%s", errorMsg))
				return
			}
			ps.selectedPackage = &info.Results[0]
			_, _, w, _ := ps.root.GetRect()
			ps.drawPackageInfo(info.Results[0], w)
		})
	}()
}

// draw package information on screen
func (ps *UI) drawPackageInfo(i InfoRecord, width int) {
	ps.details.Clear()
	ps.details.SetTitle(" [#00dfff][::b]"+i.Name+" ").SetBorderPadding(1, 1, 1, 1)
	r := 0
	ln := 0

	fields, order := getDetailFields(i)
	for _, k := range order {
		//_, _, w, _ := ps.details.GetInnerRect()
		if v, ok := fields[k]; ok {
			if ln == 1 || ln == len(fields)-1 {
				r++
			}
			// split lines if they do not fit on the screen
			w := width - (int(float64(width)*0.4) + 21) // left box = 40% size, then we use 13 characters for column 0, 2 for padding and 6 for borders
			lines := tview.WordWrap(v, w)
			ps.details.SetCell(r, 0, tview.NewTableCell(k).SetAttributes(tcell.AttrBold))
			for _, l := range lines {
				ps.details.SetCellSimple(r, 1, l)
				r++
			}
			ln++
		}
	}
	ps.details.ScrollToBeginning()
}

// gets packages from repos/AUR and displays them
func (ps *UI) showPackages(text string) {
	go func() {
		ps.locker.Lock()
		defer ps.locker.Unlock()
		defer ps.showPackageInfo(1, 0)
		packages, err := searchRepos(ps.alpmHandle, text, ps.conf.SearchMode, ps.conf.MaxResults)
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.showMessage(err.Error(), true)
			})
		}
		aurPackages, err := searchAur(ps.conf.AurRpcUrl, text, ps.conf.AurTimeout, ps.conf.SearchMode, ps.conf.MaxResults)
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.showMessage(err.Error(), true)
			})
		}

		for i := 0; i < len(aurPackages); i++ {
			aurPackages[i].IsInstalled = isInstalled(ps.alpmHandle, aurPackages[i].Name)
		}

		packages = append(packages, aurPackages...)

		sort.Slice(packages, func(i, j int) bool {
			return packages[i].Name < packages[j].Name
		})

		if len(packages) > ps.conf.MaxResults {
			packages = packages[:ps.conf.MaxResults]
		}

		// draw packages
		ps.app.QueueUpdateDraw(func() {
			if text != ps.search.GetText() {
				return
			}
			ps.drawPackages(packages)
		})
	}()
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
			Text:  pkg.Source,
			Color: color,
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
	ps.packages.SetCell(0, 0, &tview.TableCell{
		Text:          "Package",
		NotSelectable: true,
		Color:         tcell.ColorYellow,
	})
	ps.packages.SetCell(0, 1, &tview.TableCell{
		Text:          "Source",
		NotSelectable: true,
		Color:         tcell.ColorYellow,
	})
	ps.packages.SetCell(0, 2, &tview.TableCell{
		Text:          "Installed",
		NotSelectable: true,
		Color:         tcell.ColorYellow,
	})
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
	ps.details.SetTitle(" [#00dfff][::b]Usage ")
	ps.details.Clear().
		SetCellSimple(0, 0, "ENTER: Search; Install or remove a selected package").
		SetCellSimple(1, 0, "TAB / CTRL+Up/Down/Right/Left: Navigate between boxes").
		SetCellSimple(2, 0, "Up/Down: Navigate within package list").
		SetCellSimple(3, 0, "CTRL+S: Open/Close settings").
		SetCellSimple(4, 0, "CTRL+H: Show these instructions").
		SetCellSimple(5, 0, "CTRL+U: Perform sysupgrade").
		SetCellSimple(7, 0, "CTRL+Q: Quit")
}

// read settings from from and saves to config file
func (ps *UI) saveSettings(defaults bool) {
	var err error
	for i := 0; i < ps.settings.GetFormItemCount(); i++ {
		item := ps.settings.GetFormItem(i)
		if input, ok := item.(*tview.InputField); ok {
			switch input.GetLabel() {
			case "AUR RPC URL: ":
				ps.conf.AurRpcUrl = input.GetText()
			case "AUR timeout (ms): ":
				ps.conf.AurTimeout, err = strconv.Atoi(input.GetText())
				if err != nil {
					ps.showMessage("Can't convert timeout value to int", true)
					return
				}
			case "AUR search delay (ms): ":
				ps.conf.AurSearchDelay, err = strconv.Atoi(input.GetText())
				if err != nil {
					ps.showMessage("Can't convert delay value to int", true)
					return
				}
			case "Pacman DB path: ":
				ps.conf.PacmanDbPath = input.GetText()
			case "Pacman config path: ":
				ps.conf.PacmanConfigPath = input.GetText()
			case "Install command: ":
				ps.conf.InstallCommand = input.GetText()
			case "Uninstall command: ":
				ps.conf.UninstallCommand = input.GetText()
			case "Upgrade command: ":
				ps.conf.SysUpgradeCommand = input.GetText()
			case "Max search results: ":
				ps.conf.MaxResults, err = strconv.Atoi(input.GetText())
				if err != nil {
					ps.showMessage("Can't convert max results value to int", true)
					return
				}
			}
		} else if dd, ok := item.(*tview.DropDown); ok {
			switch dd.GetLabel() {
			case "Search mode: ":
				_, ps.conf.SearchMode = dd.GetCurrentOption()
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
}

// starts the spinner
func (ps *UI) startSpin() {
	go func() {
		for {
			select {
			case <-ps.quitSpin:
				return
			default:
				ms := time.Duration(60)
				ps.app.QueueUpdateDraw(func() {
					ps.spinner.SetText("/")
				})
				time.Sleep(ms * time.Millisecond)
				ps.app.QueueUpdateDraw(func() {
					ps.spinner.SetText("-")
				})
				time.Sleep(ms * time.Millisecond)
				ps.app.QueueUpdateDraw(func() {
					ps.spinner.SetText("\\")
				})
				time.Sleep(ms * time.Millisecond)
				ps.app.QueueUpdateDraw(func() {
					ps.spinner.SetText("|")
				})
			}
		}
	}()
	return
}

// installs or removes a package
func (ps *UI) installPackage() {
	row, _ := ps.packages.GetSelection()
	pkg := ps.packages.GetCell(row, 0).Text
	installed := ps.packages.GetCell(row, 2).Text

	command := ps.conf.InstallCommand
	if installed == "Y" {
		command = ps.conf.UninstallCommand
	}

	com := strings.Split(command, " ")[0]
	args := strings.Split(command, " ")[1:]
	args = append(args, pkg)

	ps.runCommand(com, args)

	// update package install status
	if isInstalled(ps.alpmHandle, pkg) {
		ps.packages.SetCellSimple(row, 2, "Y")
	} else {
		ps.packages.SetCellSimple(row, 2, "-")
	}
}

// suspends UI and runs a command in the terminal
func (ps *UI) runCommand(command string, args []string) {
	// suspend gui and run command in terminal
	ps.app.Suspend(func() {

		cmd := exec.Command(command, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// handle SIGINT and forward to the child process
		quit := handleSigint(cmd)
		cmd.Run()
		quit <- true
	})
	// we need to reinitialize the alpm handler to get the proper install state
	err := ps.reinitPacmanDbs()
	if err != nil {
		ps.showMessage(err.Error(), true)
		return
	}
}

// issues "pacman -Syu"
func (ps *UI) performSyncSysUpgrade() {
	com := strings.Split(ps.conf.SysUpgradeCommand, " ")[0]
	args := strings.Split(ps.conf.SysUpgradeCommand, " ")[1:]

	ps.runCommand(com, args)
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

// stops the spinner
func (ps *UI) stopSpin() {
	ps.quitSpin <- true
}

// re-initializes the alpm handler
func (ps *UI) reinitPacmanDbs() error {
	err := ps.alpmHandle.Release()
	if err != nil {
		return err
	}
	ps.alpmHandle, err = initPacmanDbs(ps.conf.PacmanDbPath, ps.conf.PacmanConfigPath)
	if err != nil {
		return err
	}
	return nil
}

// composes a map with fields and values (package information) for our details box
func getDetailFields(i InfoRecord) (map[string]string, []string) {
	order := []string{
		"[#1793d1]Description",
		"[#1793d1]Version",
		"[#1793d1]Licenses",
		"[#1793d1]Maintainer",
		"[#1793d1]Dependencies",
		"[#1793d1]URL",
		"[#1793d1]Votes",
		"[#1793d1]Popularity",
		"[#1793d1]Last modified",
	}

	fields := map[string]string{}
	fields[order[0]] = i.Description
	fields[order[1]] = i.Version
	fields[order[2]] = strings.Join(i.License, ", ")
	fields[order[3]] = i.Maintainer

	mdeps := strings.Join(i.MakeDepends, " (make), ")
	if mdeps != "" {
		mdeps = "\n" + mdeps
		mdeps += " (make)"
	}
	odeps := strings.Join(i.OptDepends, " (opt), ")
	if odeps != "" {
		odeps = "\n" + odeps
		odeps += " (opt)"
	}
	fields[order[4]] = strings.Join(i.Depends, ", ") + mdeps + odeps
	fields[order[5]] = i.URL
	if i.Source == "AUR" {
		fields[order[6]] = fmt.Sprintf("%d", i.NumVotes)
		fields[order[7]] = fmt.Sprintf("%f", i.Popularity)
	}
	fields[order[8]] = time.Unix(int64(i.LastModified), 0).UTC().Format("2006-01-02 - 15:04:05 (UTC)")

	return fields, order
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
			return
		}
	}()
	return quit
}
