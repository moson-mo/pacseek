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

	root    *tview.Flex
	left    *tview.Flex
	topleft *tview.Flex
	right   *tview.Flex
	bottom  *tview.Flex

	search   *tview.InputField
	packages *tview.Table
	details  *tview.Table
	spinner  *tview.TextView
	settings *tview.Form
	status   *tview.TextView

	locker     *sync.RWMutex
	spinLocker *sync.RWMutex

	quitSpin       chan bool
	requestRunning bool
	requestNumber  int
}

// New creates a UI object and makes sure everything is initialized
func New(config *config.Settings) (*UI, error) {
	ui := UI{
		conf:     config,
		app:      tview.NewApplication(),
		locker:   &sync.RWMutex{},
		quitSpin: make(chan bool),
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
func (ps *UI) Start() error {
	return ps.app.SetRoot(ps.root, true).EnableMouse(true).Run()
}

// sets up all our ui components
func (ps *UI) setupComponents() {
	// flex grids
	ps.root = tview.NewFlex()
	ps.left = tview.NewFlex().SetDirection(tview.FlexRow)
	ps.topleft = tview.NewFlex()
	ps.right = tview.NewFlex().SetDirection(tview.FlexRow)
	ps.bottom = tview.NewFlex()

	// components
	ps.search = tview.NewInputField()
	ps.packages = tview.NewTable()
	ps.details = tview.NewTable()
	ps.spinner = tview.NewTextView()
	ps.settings = tview.NewForm()
	ps.status = tview.NewTextView()

	// component config
	ps.root.SetBorder(true).
		SetTitle(" [#00dfff]pacseek - v0.1.2 ").
		SetTitleAlign(tview.AlignLeft)
	ps.search.SetLabelStyle(tcell.StyleDefault.Attributes(tcell.AttrBold)).
		SetFieldBackgroundColor(tcell.ColorDarkBlue).
		SetBorder(true)
	ps.details.SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 1, 1)
	ps.packages.SetSelectable(true, false).
		SetFixed(1, 1).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)
	ps.spinner.SetText("|").
		SetBorder(true)
	ps.bottom.AddItem(
		tview.NewTextView().
			SetText("CTRL+Q = Quit; "+
				"ENTER = Search/Install/Uninstall "+
				"TAB = Change focus; "+
				"CTRL-S = Settings"), 0, 1, false).
		SetBorder(true)
	ps.settings.SetBorder(true).
		SetTitle(" [#00dfff]Settings ").
		SetTitleAlign(tview.AlignLeft)
	ps.status.SetBorder(true)

	// settings component
	ps.setupSettingsForm()

	// layouting
	ps.root.AddItem(ps.left, 0, 40, true)
	ps.root.AddItem(ps.right, 0, 100, false)
	ps.left.AddItem(ps.topleft, 3, 1, true)
	ps.topleft.AddItem(ps.search, 0, 1, true)
	ps.topleft.AddItem(ps.spinner, 3, 1, false)
	ps.left.AddItem(ps.packages, 0, 1, false)
	ps.left.AddItem(ps.status, 3, 1, false)
	ps.right.AddItem(ps.details, 0, 1, false)
	ps.right.AddItem(ps.bottom, 3, 1, false)

	// handlers / key bindings

	// app / global
	ps.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlQ {
			ps.alpmHandle.Release()
			ps.app.Stop()
		}
		if event.Key() == tcell.KeyCtrlS {
			if ps.right.GetItem(0) != ps.settings {
				ps.right.RemoveItem(ps.details)
				ps.right.RemoveItem(ps.bottom)
				ps.right.AddItem(ps.settings, 0, 1, false)
				ps.right.AddItem(ps.bottom, 3, 1, false)
			} else {
				ps.right.RemoveItem(ps.settings)
				ps.right.RemoveItem(ps.bottom)
				ps.right.AddItem(ps.details, 0, 1, false)
				ps.right.AddItem(ps.bottom, 3, 1, false)
				ps.app.SetFocus(ps.search)
			}
		}
		return event
	})
	// search
	ps.search.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			ps.app.SetFocus(ps.packages)
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			ps.showPackages(ps.search.GetText())
		}
		return event
	})

	// packages
	ps.packages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			if ps.right.GetItem(0) == ps.settings {
				ps.app.SetFocus(ps.settings.GetFormItem(0))
			} else {
				ps.app.SetFocus(ps.search)
			}
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
		for i := 0; i < ps.settings.GetFormItemCount(); i++ {
			item := ps.settings.GetFormItem(i)
			if i+1 < ps.settings.GetFormItemCount() {
				next := ps.settings.GetFormItem(i + 1)
				if input, ok := item.(*tview.InputField); ok {
					input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyTAB {
							ps.app.SetFocus(next)
							return nil
						}
						return event
					})
				} else if dd, ok := item.(*tview.DropDown); ok {
					dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyTAB {
							ps.app.SetFocus(next)
							return nil
						}
						return event
					})
				}
			} else {
				if input, ok := item.(*tview.InputField); ok {
					input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyTAB {
							ps.app.SetFocus(ps.settings.GetButton(0))
							return nil
						}
						return event
					})
				} else if dd, ok := item.(*tview.DropDown); ok {
					dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyTAB {
							ps.app.SetFocus(ps.settings.GetButton(0))
							return nil
						}
						return event
					})
				}
			}
		}

		ps.settings.GetButton(0).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTAB {
				ps.app.SetFocus(ps.settings.GetButton(1))
				return nil
			}
			return event
		})
		ps.settings.GetButton(1).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTAB {
				ps.app.SetFocus(ps.search)
				return nil
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
						ps.status.SetText("Can't convert timeout value to int")
						return
					}
				case "AUR search delay (ms): ":
					ps.conf.AurSearchDelay, err = strconv.Atoi(input.GetText())
					if err != nil {
						ps.status.SetText("Can't convert delay value to int")
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
						ps.status.SetText("Can't convert max results value to int")
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
			ps.status.SetText(err.Error())
			return
		}
		ps.status.SetText("Settings have been saved")
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
			ps.details.SetTitle(" [#1793d1]" + pkg + " - Retrieving data... ")
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
			r := info.Results[0]

			ps.details.SetTitle(" [#00dfff]"+r.Name+" ").SetBorderPadding(1, 1, 1, 1)
			ps.details.SetCell(0, 0, tview.NewTableCell("[#1793d1]Description").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(0, 1, r.Description)
			ps.details.SetCell(1, 0, tview.NewTableCell("[#1793d1]Version").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(1, 1, r.Version)
			ps.details.SetCell(2, 0, tview.NewTableCell("[#1793d1]Licenses").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(2, 1, strings.Join(r.License, ", "))
			ps.details.SetCell(3, 0, tview.NewTableCell("[#1793d1]Maintainer").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(3, 1, r.Maintainer)
			if source == "AUR" {
				ps.details.SetCell(4, 0, tview.NewTableCell("[#1793d1]Votes").SetAttributes(tcell.AttrBold))
				ps.details.SetCellSimple(4, 1, fmt.Sprintf("%d", r.NumVotes))
				ps.details.SetCell(5, 0, tview.NewTableCell("[#1793d1]Popularity").SetAttributes(tcell.AttrBold))
				ps.details.SetCellSimple(5, 1, fmt.Sprintf("%f", r.Popularity))
			}
			ps.details.SetCell(6, 0, tview.NewTableCell("[#1793d1]Dependencies").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(6, 1, strings.Join(r.Depends, ", "))
			ps.details.SetCell(7, 0, tview.NewTableCell("[#1793d1]URL").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(7, 1, r.URL)
			ps.details.SetCell(8, 0, tview.NewTableCell("[#1793d1]Last modified").SetAttributes(tcell.AttrBold))
			ps.details.SetCellSimple(8, 1, time.Unix(int64(r.LastModified), 0).UTC().Format("2006-01-02 - 15:04:05 (UTC)"))
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
				ps.status.SetText(err.Error())
			})
		}
		aurPackages, err := searchAur(ps.conf.AurRpcUrl, text, ps.conf.AurTimeout, ps.conf.SearchMode, ps.conf.MaxResults)
		if err != nil {
			ps.app.QueueUpdateDraw(func() {
				ps.status.SetText(err.Error())
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

// starts the spinner
func (ps *UI) startSpin() {
	go func() {
		for {
			select {
			case <-ps.quitSpin:
				return
			default:
				ps.app.QueueUpdateDraw(func() {
					txt := ps.spinner.GetText(true)
					if txt == "|" {
						ps.spinner.SetText("/")
					} else if txt == "/" {
						ps.spinner.SetText("-")
					} else if txt == "-" {
						ps.spinner.SetText("\\")
					} else if txt == "\\" {
						ps.spinner.SetText("|")
					} else {
						ps.spinner.SetText("|")
					}
				})
				time.Sleep(200 * time.Millisecond)
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
			ps.status.SetText(err.Error())
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
