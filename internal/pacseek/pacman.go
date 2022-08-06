package pacseek

import (
	"errors"
	"math"
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
func searchRepos(h *alpm.Handle, term string, mode string, by string, maxResults int, localOnly bool) ([]Package, error) {
	packages := []Package{}
	if h == nil {
		return packages, errors.New("alpm handle is nil")
	}
	dbs, err := h.SyncDBs()
	if err != nil {
		return packages, err
	}
	local, err := h.LocalDB()
	if err != nil {
		return packages, err
	}

	searchDbs := dbs.Slice()
	if localOnly {
		searchDbs = []alpm.IDB{local}
	}

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
				installed := false
				if local.Pkg(pkg.Name()) != nil {
					installed = true
				}
				packages = append(packages, Package{
					Name:         pkg.Name(),
					Source:       db.Name(),
					IsInstalled:  installed,
					LastModified: int(pkg.BuildDate().Unix()),
					Popularity:   math.MaxFloat64,
				})
				counter++
			}
		}
	}
	return packages, nil
}

// checks the local db if a package is installed
func isPackageInstalled(h *alpm.Handle, pkg string) bool {
	local, err := h.LocalDB()
	if err != nil {
		return false
	}
	local.SetUsage(alpm.UsageSearch)

	return local.Pkg(pkg) != nil
}

// retrieves package information from the pacman DB's and returns it in the same format as the AUR call
func infoPacman(h *alpm.Handle, pkgs []string) RpcResult {
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
				RequiredBy:   p.ComputeRequiredBy(),
				PackageBase:  p.Base(),
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
