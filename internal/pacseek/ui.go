package pacseek

import (
	"io"
	"sync"
	"time"

	"github.com/Jguer/go-alpm/v2"
	"github.com/moson-mo/pacseek/internal/config"
	"github.com/moson-mo/pacseek/internal/util"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

const (
	UrlAurPackage   = "https://aur.archlinux.org/packages/%s"
	UrlAurPkgbuild  = "https://raw.githubusercontent.com/archlinux/aur/%s/PKGBUILD"
	UrlPackage      = "https://archlinux.org/packages/%s/%s/%s"
	UrlRepoPkgbuild = "https://raw.githubusercontent.com/archlinux/svntogit-%s/packages/%s/trunk/PKGBUILD"

	version = "1.5.1"
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

	pkgbuildWriter io.Writer
}

// New creates a UI object and makes sure everything is initialized
func New(conf *config.Settings, repos []string, asciiMode, monoMode bool) (*UI, error) {
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

		filterRepos:   repos,
		asciiMode:     asciiMode,
		sortAscending: true,
	}

	// get users default shell
	ui.shell = util.Shell()

	// get a handle to the pacman DB's
	var err error
	ui.alpmHandle, err = initPacmanDbs(conf.PacmanDbPath, conf.PacmanConfigPath, repos)
	if err != nil {
		return nil, err
	}

	// setup UI
	ui.createComponents()
	if monoMode {
		ui.conf.SetColorScheme("Monochrome")
	}
	if asciiMode {
		ui.applyASCIIMode()
	}
	ui.applyColors()
	ui.setupKeyBindings()
	ui.setupSettingsForm()

	return &ui, nil
}

// Start runs application / event-loop
func (ps *UI) Start(term string) error {
	if term != "" {
		ps.inputSearch.SetText(term)
		ps.displayPackages(term)
	}
	return ps.app.SetRoot(ps.flexRoot, true).EnableMouse(true).Run()
}

// getArchRepos returns a list of Arch Linux repositories
func getArchRepos() []string {
	return []string{"core", "community", "community-testing", "extra", "kde-unstable", "multilib", "multilib-testing", "testing"}
}
