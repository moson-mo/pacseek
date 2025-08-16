package pacseek

// SearchResults is a data structure that is being sent back from the RPC service
type SearchResults struct {
	Error       string       `json:"error,omitempty"`
	Resultcount int          `json:"resultcount"`
	Results     []InfoRecord `json:"results"`
	Type        string       `json:"type"`
	Version     int          `json:"version"`
}

// InfoRecord is a data structure for "search" API calls (results)
type InfoRecord struct {
	CheckDepends      []string `json:"CheckDepends,omitempty"`
	Conflicts         []string `json:"Conflicts,omitempty"`
	Depends           []string `json:"Depends,omitempty"`
	Description       string   `json:"Description"`
	FirstSubmitted    int      `json:"FirstSubmitted"`
	Groups            []string `json:"Groups,omitempty"`
	ID                int      `json:"ID"`
	Keywords          []string `json:"Keywords"`
	LastModified      int      `json:"LastModified"`
	License           []string `json:"License"`
	Maintainer        string   `json:"Maintainer"`
	MakeDepends       []string `json:"MakeDepends,omitempty"`
	Name              string   `json:"Name"`
	NumVotes          int      `json:"NumVotes"`
	OptDepends        []string `json:"OptDepends,omitempty"`
	OutOfDate         int      `json:"OutOfDate"`
	PackageBase       string   `json:"PackageBase"`
	PackageBaseID     int      `json:"PackageBaseID"`
	Popularity        float64  `json:"Popularity"`
	Provides          []string `json:"Provides,omitempty"`
	Replaces          []string `json:"Replaces,omitempty"`
	RequiredBy        []string `json:"RequiredBy,omitempty"`
	URL               string   `json:"URL"`
	URLPath           string   `json:"URLPath"`
	Version           string   `json:"Version"`
	LocalVersion      string
	Source            string `json:"Source"`
	Architecture      string `json:"Architecture"`
	IsIgnored         bool
	DepsAndSatisfiers []DependencySatisfier
}

type PkgState int8

const (
	PkgNone      PkgState = 0x0
	PkgInstalled          = 0x1
	PkgMarked             = 0x2
)

type PkgStatus struct {
	Pkg InfoRecord
	//Name string
	//ID int
	//Source string
	State PkgState
}

var pkglist []PkgStatus

type DependencySatisfier struct {
	DepType   string
	DepName   string
	Satisfier string
	Installed bool
}

// Package is a data structure for the package tview table
type Package struct {
	Name         string
	Source       string
	IsInstalled  bool
	LastModified int
	Popularity   float64
}

// get package information
func (ps *UI) getInfo(source string, pkgs ...string) SearchResults {
	sr := SearchResults{}
	if source == "AUR" || source == "all" {
		sr = infoAur(ps.conf.AurRpcUrl, ps.conf.AurTimeout, pkgs...)
		if source == "all" {
			sr.Results = append(sr.Results, infoPacman(ps.alpmHandle, ps.conf.ComputeRequiredBy, pkgs...).Results...)
		}
	} else {
		sr = infoPacman(ps.alpmHandle, ps.conf.ComputeRequiredBy, pkgs...)
	}

	addLocalSatisfiers(ps.alpmHandle, sr.Results...)
	return sr
}
