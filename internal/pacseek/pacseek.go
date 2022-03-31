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

	search   *tview.InputField
	packages *tview.Table
	details  *tview.Table
	spinner  *tview.TextView
	settings *tview.Form
	status   *tview.TextView

	locker        *sync.RWMutex
	messageLocker *sync.RWMutex

	quitSpin       chan bool
	requestRunning bool
	requestNumber  int
}

// New creates a UI object and makes sure everything is initialized
func New(config *config.Settings) (*UI, error) {
	ui := UI{
		conf:          config,
		app:           tview.NewApplication(),
		locker:        &sync.RWMutex{},
		messageLocker: &sync.RWMutex{},
		quitSpin:      make(chan bool),
	}
	ui.setupComponents()

	var err error
	ui.alpmHandle, err = initPacmanDbs(config.PacmanDbPath, config.PacmanConfigPath)
	if err != nil {
		return nil, err
	}

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
		SetTitle(" [#00dfff][::b]pacseek - v0.2.1 ").
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

	// settings component
	ps.setupSettingsForm()

	// layouting
	ps.root.AddItem(ps.container, 0, 1, true)
	ps.root.AddItem(ps.status, 0, 0, false)
	ps.container.AddItem(ps.left, 0, 50, true)
	ps.container.AddItem(ps.right, 0, 100, false)
	ps.left.AddItem(ps.topleft, 3, 1, true)
	ps.topleft.AddItem(ps.search, 0, 1, true)
	ps.topleft.AddItem(ps.spinner, 3, 1, false)
	ps.left.AddItem(ps.packages, 0, 1, false)
	ps.right.AddItem(ps.details, 0, 1, false)

	// handlers / key bindings

	// app / global
	ps.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlQ {
			ps.alpmHandle.Release()
			ps.app.Stop()
		}
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
		if event.Key() == tcell.KeyCtrlH {
			ps.showHelp()
			if ps.right.GetItem(0) == ps.settings {
				ps.right.Clear()
				ps.right.AddItem(ps.details, 0, 1, false)
			}
			return nil
		}
		return event
	})

	// search
	ps.search.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB || event.Key() == tcell.KeyDown {
			ps.app.SetFocus(ps.packages)
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			ps.showPackages(ps.search.GetText())
			return nil
		}
		if event.Key() == tcell.KeyRight && event.Modifiers() == tcell.ModCtrl && ps.right.GetItem(0) == ps.settings {
			ps.app.SetFocus(ps.settings)
			return nil
		}

		return event
	})

	// packages
	ps.packages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
		if event.Key() == tcell.KeyRight && ps.right.GetItem(0) == ps.settings {
			ps.app.SetFocus(ps.settings.GetFormItem(0))
			return nil
		}
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

		// input fields
		ps.settings.AddInputField("AUR RPC URL: ", ps.conf.AurRpcUrl, 40, nil, nil)
		ps.settings.AddInputField("AUR timeout (ms): ", strconv.Itoa(ps.conf.AurTimeout), 6, nil, nil)
		ps.settings.AddInputField("AUR search delay (ms): ", strconv.Itoa(ps.conf.AurSearchDelay), 6, nil, nil)
		ps.settings.AddInputField("Max search results: ", strconv.Itoa(ps.conf.MaxResults), 6, nil, nil)
		ps.settings.AddInputField("Pacman DB path: ", ps.conf.PacmanDbPath, 40, nil, nil)
		ps.settings.AddInputField("Pacman config path: ", ps.conf.PacmanConfigPath, 40, nil, nil)
		ps.settings.AddInputField("Install command: ", ps.conf.InstallCommand, 40, nil, nil)
		ps.settings.AddInputField("Uninstall command: ", ps.conf.UninstallCommand, 40, nil, nil)
		ps.settings.AddDropDown("Search mode: ", []string{"StartsWith", "Contains"}, mode, nil)

		// key bindings
		ps.settings.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyLeft && event.Modifiers() == tcell.ModCtrl {
				ps.app.SetFocus(ps.packages)
				return nil
			}
			return event
		})

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

	// buttons
	ps.settings.AddButton("Save", func() {
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
		ps.settings.GetButton(0).SetLabel("Saved")
		err = ps.conf.Save()
		if err != nil {
			ps.showMessage(err.Error(), true)
			return
		}
		ps.showMessage("Settings have been saved", false)
	})

	ps.settings.AddButton("Defaults", func() {
		ps.conf = config.Defaults()
		ps.settings.Clear(false)
		addFields()
	})

	// add our input fields
	addFields()
}

// retrieves package information from repo/AUR and displays them
func (ps *UI) showPackageInfo(row, column int) {
	if row == -1 || row+1 > ps.packages.GetRowCount() {
		return
	}
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
			i := info.Results[0]

			ps.details.SetTitle(" [#00dfff][::b]"+i.Name+" ").SetBorderPadding(1, 1, 1, 1)
			r := 0
			ln := 0

			fields, order := getDetailFields(i, source)
			for _, k := range order {
				_, _, w, _ := ps.details.GetInnerRect()
				if v, ok := fields[k]; ok {
					if ln == 1 || ln == len(fields)-1 {
						r++
					}
					// split lines if they do not fit on the screen
					lines := tview.WordWrap(v, w-15) // we use 13 characters for column 0 (+ 2 chars for padding)
					ps.details.SetCell(r, 0, tview.NewTableCell(k).SetAttributes(tcell.AttrBold))
					for _, l := range lines {
						ps.details.SetCellSimple(r, 1, l)
						r++
					}
					ln++
				}
			}
		})
	}()
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
			ps.packages.Clear()

			// header
			ps.addPackagesHeader()

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
					Text:      pkg.Source,
					Expansion: 1000,
					Color:     color,
				})
				ps.packages.SetCellSimple(i+1, 2, installed)
			}
			ps.packages.ScrollToBeginning()
		})
	}()
}

// adds header row to package table
func (ps *UI) addPackagesHeader() {
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
		SetCellSimple(6, 0, "CTRL+Q: Quit")
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

	// suspend gui and run command in terminal
	ps.app.Suspend(func() {
		com := strings.Split(command, " ")[0]
		args := strings.Split(command, " ")[1:]
		args = append(args, pkg)

		cmd := exec.Command(com, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// handle SIGINT and forward to the child process
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			<-c
			if cmd != nil {
				cmd.Process.Signal(os.Interrupt)
			}
		}()

		cmd.Run()

		// we need to reinitialize the alpm handler to get the proper install state
		err := ps.reinitPacmanDbs()
		if err != nil {
			ps.showMessage(err.Error(), true)
			return
		}

		// update package install status
		if isInstalled(ps.alpmHandle, pkg) {
			ps.packages.SetCellSimple(row, 2, "Y")
		} else {
			ps.packages.SetCellSimple(row, 2, "-")
		}

	})
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
func getDetailFields(i InfoRecord, source string) (map[string]string, []string) {
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
		mdeps += " (make)"
	}
	fields[order[4]] = strings.Join(i.Depends, ", ") + mdeps
	fields[order[5]] = i.URL
	if source == "AUR" {
		fields[order[6]] = fmt.Sprintf("%d", i.NumVotes)
		fields[order[7]] = fmt.Sprintf("%f", i.Popularity)
	}
	fields[order[8]] = time.Unix(int64(i.LastModified), 0).UTC().Format("2006-01-02 - 15:04:05 (UTC)")

	return fields, order
}
