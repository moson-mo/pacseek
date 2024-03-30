package pacseek

import (
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/Jguer/go-alpm/v2"
	"github.com/moson-mo/pacseek/internal/args"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

const (
	UrlAurPackage   = "https://aur.archlinux.org/packages/%s"
	UrlAurPkgbuild  = "https://raw.githubusercontent.com/archlinux/aur/%s/PKGBUILD"
	UrlPackage      = "https://archlinux.org/packages/%s/%s/%s"
	UrlArmPackage   = "https://archlinuxarm.org/packages/%s/%s"
	UrlRepoPkgbuild = "https://gitlab.archlinux.org/archlinux/packaging/packages/%s/-/raw/main/PKGBUILD"

	UrlAurMaintainer = "https://aur.archlinux.org/packages?SeB=m&K=%s"

	version = "1.8.3"
)

// UI is holding our application information and all tview components
type UI struct {
	conf *config.Settings
	app  *tview.Application

	alpmHandle *alpm.Handle

	flexRoot      *tview.Flex
	flexLeft      *tview.Flex
	flexTopLeft   *tview.Flex
	flexRight     *tview.Flex
	flexContainer *tview.Flex

	inputSearch   *tview.InputField
	tablePackages *tview.Table
	tableDetails  *tview.Table
	spinner       *tview.TextView
	formSettings  *tview.Form
	textMessage   *tview.TextView
	textPkgbuild  *tview.TextView
	prevComponent tview.Primitive
	tableNews     *tview.Table

	locker        *sync.RWMutex
	messageLocker *sync.RWMutex

	quitSpin        chan bool
	width           int
	leftProportion  int
	selectedPackage *InfoRecord
	settingsChanged bool
	cacheInfo       *cache.Cache
	cacheSearch     *cache.Cache
	cachePkgbuild   *cache.Cache
	filterRepos     []string
	asciiMode       bool
	shell           string
	lastSearchTerm  string
	shownPackages   []Package
	sortAscending   bool
	isArm           bool
	flags           args.Flags

	tableDetailsMore bool

	pkgbuildWriter io.Writer
}

// New creates a UI object and makes sure everything is initialized
func New(conf *config.Settings, flags args.Flags) (*UI, error) {
	ui := UI{
		conf:            conf,
		app:             tview.NewApplication(),
		locker:          &sync.RWMutex{},
		messageLocker:   &sync.RWMutex{},
		quitSpin:        make(chan bool),
		settingsChanged: false,
		cacheInfo:       cache.New(time.Duration(conf.CacheExpiry)*time.Minute, 1*time.Minute),
		cacheSearch:     cache.New(time.Duration(conf.CacheExpiry)*time.Minute, 1*time.Minute),
		cachePkgbuild:   cache.New(time.Duration(conf.CacheExpiry)*time.Minute, 1*time.Minute),

		flags:         flags,
		sortAscending: true,
		isArm:         runtime.GOARCH != "amd64",
	}

	// get users default shell
	ui.shell = util.Shell()

	// get a handle to the pacman DB's
	var err error
	ui.alpmHandle, err = initPacmanDbs(conf.PacmanDbPath, conf.PacmanConfigPath, flags.Repositories)
	if err != nil {
		return nil, err
	}

	// set window layout
	if conf.SaveWindowLayout {
		if conf.LeftProportion < 1 || conf.LeftProportion > 9 {
			conf.LeftProportion = 4
		}
		ui.leftProportion = conf.LeftProportion
	} else {
		ui.leftProportion = 4
	}

	// setup UI
	ui.createComponents()
	if flags.MonochromeMode {
		ui.conf.SetColorScheme("Monochrome")
		ui.conf.SetTransparency(ui.conf.Transparent)
	}
	if flags.AsciiMode {
		ui.applyASCIIMode()
	}

	ui.applyColors()
	ui.applyGlyphStyle()
	ui.setupKeyBindings()
	ui.setupSettingsForm()

	return &ui, nil
}

// Start runs application / event-loop
func (ps *UI) Start() error {
	if ps.flags.SearchTerm != "" {
		ps.inputSearch.SetText(ps.flags.SearchTerm)
		ps.displayPackages(ps.flags.SearchTerm)
	} else {
		if ps.flags.ShowInstalled {
			ps.displayInstalled(ps.flags.ShowUpdates)
		}
		if ps.flags.ShowUpdates && !ps.flags.ShowInstalled {
			ps.displayUpgradable()
		}
	}

	return ps.app.SetRoot(ps.flexRoot, true).EnableMouse(true).Run()
}

// getArchRepos returns a list of Arch Linux repositories
func getArchRepos() []string {
	return []string{"core", "core-testing", "extra", "extra-testing", "multilib", "multilib-testing", "kde-unstable", "gnome-unstable"}
}

// getArchArmRepos returns a list of Arch Linux ARM repositories
func getArchArmRepos() []string {
	return []string{"core", "community", "extra", "aur", "alarm"}
}
