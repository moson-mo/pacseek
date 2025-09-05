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
	h.SetIgnorePkgs(conf.IgnorePkg)
	h.SetIgnoreGroups(conf.IgnoreGroup)

	return h, nil
}

// searches the pacman databases and returns packages that could be found (starting with "term")
func searchRepos(h *alpm.Handle, term string, by string, maxResults int) ([]Package, []Package, error) {
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

			if util.IsMatch(pkg.Name(), term) ||
				(by == "Name & Description" && util.IsMatch(strings.ToLower(pkg.Description()), term)) {
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

func suggestRepos(h *alpm.Handle, term string) []string {
	pkgs, _, _ := searchRepos(h, term, "", 20)

	names := []string{}
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}

	return names
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

	results := infoPacman(h, computeRequiredBy, installed...).Results
	addLocalSatisfiers(h, results...)
	return results, notFound
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
func infoPacman(h *alpm.Handle, computeRequiredBy bool, pkgs ...string) SearchResults {
	r := SearchResults{
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
			cdeps := []string{}
			prov := []string{}
			conf := []string{}

			for _, d := range p.Depends().Slice() {
				deps = append(deps, d.String())
			}
			for _, d := range p.MakeDepends().Slice() {
				makedeps = append(makedeps, d.String())
			}
			for _, d := range p.OptionalDepends().Slice() {
				odeps = append(odeps, d.String())
			}
			for _, d := range p.CheckDepends().Slice() {
				cdeps = append(cdeps, d.String())
			}
			for _, pr := range p.Provides().Slice() {
				prov = append(prov, pr.String())
			}
			for _, c := range p.Conflicts().Slice() {
				conf = append(conf, c.String())
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
				CheckDepends: cdeps,
				URL:          p.URL(),
				LastModified: int(p.BuildDate().UTC().Unix()),
				Source:       db.Name(),
				Architecture: p.Architecture(),
				PackageBase:  p.Base(),
				IsIgnored:    p.ShouldIgnore(),
			}

			if computeRequiredBy {
				optFor := p.ComputeOptionalFor()
				for i, pkg := range optFor {
					optFor[i] = pkg + " (opt)"
				}
				i.RequiredBy = append(p.ComputeRequiredBy(), optFor...)
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

// add locally installed satisfiers to package info records
func addLocalSatisfiers(h *alpm.Handle, pkgs ...InfoRecord) {
	local, err := h.LocalDB()

	for i := 0; i < len(pkgs); i++ {
		depList := []struct {
			deptype string
			deps    []string
		}{
			{"dep", pkgs[i].Depends},
			{"opt", pkgs[i].OptDepends},
			{"make", pkgs[i].MakeDepends},
			{"check", pkgs[i].CheckDepends},
		}

		satisfiers := []DependencySatisfier{}
		for _, entry := range depList {
			for _, dep := range entry.deps {
				sat := DependencySatisfier{
					DepName:   dep,
					DepType:   entry.deptype,
					Installed: false,
				}
				if err == nil {
					found, _ := local.PkgCache().FindSatisfier(dep)
					if found != nil {
						sat.Satisfier = found.Name()
						sat.Installed = true
					}
				}
				satisfiers = append(satisfiers, sat)
			}
		}
		pkgs[i].DepsAndSatisfiers = satisfiers
	}
}
