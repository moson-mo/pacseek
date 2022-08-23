package pacseek

import (
	"errors"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/Jguer/go-alpm/v2"
	pconf "github.com/Morganamilo/go-pacmanconf"
	"github.com/moson-mo/pacseek/internal/util"
)

// creates the alpm handler used to search packages
func initPacmanDbs(dbPath, confPath string, repos []string) (*alpm.Handle, error) {
	h, err := alpm.Initialize("/", dbPath)
	if err != nil {
		return nil, err
	}

	conf, _, err := pconf.ParseFile(confPath)
	if err != nil {
		return nil, err
	}

	for _, repo := range conf.Repos {
		if (len(repos) > 0 && util.SliceContains(repos, repo.Name)) || len(repos) == 0 {
			_, err := h.RegisterSyncDB(repo.Name, 0)
			if err != nil {
				return nil, err
			}
		}
	}
	return h, nil
}

// searches the pacman databases and returns packages that could be found (starting with "term")
func searchRepos(h *alpm.Handle, term string, mode string, by string, maxResults int) ([]Package, []Package, error) {
	packages := []Package{}
	installed := []Package{}

	if h == nil {
		return packages, installed, errors.New("alpm handle is nil")
	}
	dbs, err := h.SyncDBs()
	if err != nil {
		return packages, installed, err
	}
	local, err := h.LocalDB()
	if err != nil {
		return packages, installed, err
	}

	searchDbs := append(dbs.Slice(), local)

	counter := 0
	for _, db := range searchDbs {
		for _, pkg := range db.PkgCache().Slice() {
			if counter >= maxResults {
				break
			}
			compFunc := strings.HasPrefix
			if mode == "Contains" {
				compFunc = strings.Contains
			}

			if compFunc(pkg.Name(), term) ||
				(by == "Name & Description" && compFunc(pkg.Description(), term)) {
				pkg := Package{
					Name:         pkg.Name(),
					Source:       db.Name(),
					IsInstalled:  local.Pkg(pkg.Name()) != nil,
					LastModified: int(pkg.BuildDate().Unix()),
					Popularity:   math.MaxFloat64,
				}
				if db != local {
					packages = append(packages, pkg)
				} else {
					installed = append(installed, pkg)
				}

				counter++
			}
		}
	}
	return packages, installed, nil
}

// create/update temporary sync DB
func syncToTempDB(confPath string, repos []string) (*alpm.Handle, error) {
	// check if fakeroot is installed
	if _, err := os.Stat("/usr/bin/fakeroot"); errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("fakeroot not installed")
	}
	conf, _, err := pconf.ParseFile(confPath)
	if err != nil {
		return nil, err
	}
	/*
		We use the same naming as "checkupdates" to have less data to transfer
		in case the user already makes use of checkupdates...
	*/
	tmpdb := os.TempDir() + "/checkup-db-" + strconv.Itoa(os.Getuid())
	local := tmpdb + "/local"

	// create directory and symlink if needed
	if _, err := os.Stat(tmpdb); errors.Is(err, fs.ErrNotExist) {
		err := os.MkdirAll(tmpdb, 0755)
		if err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(local); errors.Is(err, fs.ErrNotExist) {
		err := os.Symlink(path.Join(conf.DBPath, "local"), local)
		if err != nil {
			return nil, err
		}
	}

	// execute pacman and sync to temporary db
	cmd := exec.Command("fakeroot", "--", "pacman", "-Sy", "--dbpath="+tmpdb)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(out))
	}

	h, err := initPacmanDbs(tmpdb, confPath, repos)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// returns packages that can be upgraded & packages that only exist locally
func getUpgradable(h *alpm.Handle, computeRequiredBy bool) ([]InfoRecord, []string) {
	upgradable := []string{}
	notFound := []string{}

	if h == nil {
		return []InfoRecord{}, notFound
	}
	dbs, err := h.SyncDBs()
	if err != nil {
		return []InfoRecord{}, notFound
	}
	local, err := h.LocalDB()
	if err != nil {
		return []InfoRecord{}, notFound
	}

	for _, lpkg := range local.PkgCache().Slice() {
		found := false
		for _, db := range dbs.Slice() {
			pkg := db.Pkg(lpkg.Name())
			if pkg != nil {
				found = true
				if alpm.VerCmp(pkg.Version(), lpkg.Version()) > 0 {
					upgradable = append(upgradable, pkg.Name())
				}
				break
			}
		}
		if !found {
			upgradable = append(upgradable, lpkg.Name())
			notFound = append(notFound, lpkg.Name())
		}
	}

	return infoPacman(h, computeRequiredBy, upgradable...).Results, notFound
}

// returns packages that can be upgraded & packages that only exist locally
func getInstalled(h *alpm.Handle, computeRequiredBy bool) ([]InfoRecord, []string) {
	installed := []string{}
	notFound := []string{}

	if h == nil {
		return []InfoRecord{}, notFound
	}
	dbs, err := h.SyncDBs()
	if err != nil {
		return []InfoRecord{}, notFound
	}
	local, err := h.LocalDB()
	if err != nil {
		return []InfoRecord{}, notFound
	}

	for _, lpkg := range local.PkgCache().Slice() {
		found := false
		for _, db := range dbs.Slice() {
			if pkg := db.Pkg(lpkg.Name()); pkg != nil {
				found = true
				installed = append(installed, pkg.Name())
				break
			}
		}
		if !found {
			installed = append(installed, lpkg.Name())
			notFound = append(notFound, lpkg.Name())
		}
	}

	return infoPacman(h, computeRequiredBy, installed...).Results, notFound
}

// checks the local db if a package is installed
func isPackageInstalled(h *alpm.Handle, pkg string) bool {
	local, err := h.LocalDB()
	if err != nil {
		return false
	}

	return local.Pkg(pkg) != nil
}

// retrieves package information from the pacman DB's and returns it in the same format as the AUR call
func infoPacman(h *alpm.Handle, computeRequiredBy bool, pkgs ...string) RpcResult {
	r := RpcResult{
		Results: []InfoRecord{},
	}

	dbs, err := h.SyncDBs()
	if err != nil {
		r.Error = err.Error()
		return r
	}

	local, err := h.LocalDB()
	if err != nil {
		r.Error = err.Error()
		return r
	}
	dbslice := append(dbs.Slice(), local)

	for _, pkg := range pkgs {
		for _, db := range dbslice {
			p := db.Pkg(pkg)
			if p == nil {
				continue
			}

			deps := []string{}
			makedeps := []string{}
			odeps := []string{}
			prov := []string{}
			conf := []string{}
			for _, d := range p.Depends().Slice() {
				deps = append(deps, d.Name)
			}
			for _, d := range p.MakeDepends().Slice() {
				makedeps = append(makedeps, d.Name)
			}
			for _, d := range p.OptionalDepends().Slice() {
				odeps = append(odeps, d.Name)
			}

			i := InfoRecord{
				Name:         p.Name(),
				Description:  p.Description(),
				Provides:     prov,
				Conflicts:    conf,
				Version:      p.Version(),
				License:      p.Licenses().Slice(),
				Maintainer:   p.Packager(),
				Depends:      deps,
				MakeDepends:  makedeps,
				OptDepends:   odeps,
				URL:          p.URL(),
				LastModified: int(p.BuildDate().UTC().Unix()),
				Source:       db.Name(),
				Architecture: p.Architecture(),
				PackageBase:  p.Base(),
			}

			if computeRequiredBy {
				i.RequiredBy = p.ComputeRequiredBy()
			}
			if lpkg := local.Pkg(p.Name()); lpkg != nil {
				i.LocalVersion = lpkg.Version()
			}
			if db.Name() == "local" {
				i.Description = p.Description() + "\n[red]* Package not found in repositories/AUR *"
			}

			r.Results = append(r.Results, i)

			break
		}
	}

	return r
}
