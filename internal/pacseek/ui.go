package pacseek

import (
	"os"
	"sync"
	"time"

	"github.com/Jguer/go-alpm/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

const (
	colorHighlight = "[#1793d1]"
	colorTitle     = "[#00dfff]"
	aurPackageUrl  = "https://aur.archlinux.org/packages/"
	packageUrl     = "https://archlinux.org/packages/"

	version = "1.3.3"
)

var archRepos = []string{"core", "community", "community-testing", "extra", "kde-unstable", "multilib", "multilib-testing", "testing"}

// UI is holding our application information and all tview components
type UI struct {
	conf  *config.Settings
	app   *tview.Application
	shell string

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
	leftProportion  int
	selectedPackage *InfoRecord
	settingsChanged bool
	infoCache       *cache.Cache
	searchCache     *cache.Cache
	repos           []string
	asciiMode       bool
}

// New creates a UI object and makes sure everything is initialized
func New(config *config.Settings, repos []string, asciiMode bool) (*UI, error) {
	ui := UI{
		conf:            config,
		app:             tview.NewApplication(),
		locker:          &sync.RWMutex{},
		messageLocker:   &sync.RWMutex{},
		quitSpin:        make(chan bool),
		settingsChanged: false,
		infoCache:       cache.New(time.Duration(config.CacheExpiry)*time.Minute, 1*time.Minute),
		searchCache:     cache.New(time.Duration(config.CacheExpiry)*time.Minute, 1*time.Minute),
		repos:           repos,
		asciiMode:       asciiMode,
	}

	// get users default shell
	ui.shell = getShell()

	// get a handle to the pacman DB's
	var err error
	ui.alpmHandle, err = initPacmanDbs(config.PacmanDbPath, config.PacmanConfigPath, repos)
	if err != nil {
		return nil, err
	}

	// setup UI
	ui.setupComponents()
	if asciiMode {
		ui.setASCIIMode()
	}
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

// get users shell
func getShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh" // fallback
	}
	return shell
}
