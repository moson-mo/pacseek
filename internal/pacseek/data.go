package pacseek

// RpcResult is a data structure that is being sent back from the RPC service
type RpcResult struct {
	Error       string       `json:"error,omitempty"`
	Resultcount int          `json:"resultcount"`
	Results     []InfoRecord `json:"results"`
	Type        string       `json:"type"`
	Version     int          `json:"version"`
}

// InfoRecord is a data structure for "search" API calls (results)
type InfoRecord struct {
	CheckDepends   []string `json:"CheckDepends,omitempty"`
	Conflicts      []string `json:"Conflicts,omitempty"`
	Depends        []string `json:"Depends,omitempty"`
	Description    string   `json:"Description"`
	FirstSubmitted int      `json:"FirstSubmitted"`
	Groups         []string `json:"Groups,omitempty"`
	ID             int      `json:"ID"`
	Keywords       []string `json:"Keywords"`
	LastModified   int      `json:"LastModified"`
	License        []string `json:"License"`
	Maintainer     string   `json:"Maintainer"`
	MakeDepends    []string `json:"MakeDepends,omitempty"`
	Name           string   `json:"Name"`
	NumVotes       int      `json:"NumVotes"`
	OptDepends     []string `json:"OptDepends,omitempty"`
	OutOfDate      int      `json:"OutOfDate"`
	PackageBase    string   `json:"PackageBase"`
	PackageBaseID  int      `json:"PackageBaseID"`
	Popularity     float64  `json:"Popularity"`
	Provides       []string `json:"Provides,omitempty"`
	Replaces       []string `json:"Replaces,omitempty"`
	RequiredBy     []string `json:"RequiredBy,omitempty"`
	URL            string   `json:"URL"`
	URLPath        string   `json:"URLPath"`
	Version        string   `json:"Version"`
	Source         string   `json:"Source"`
	Architecture   string   `json:"Architecture"`
}

// Package is a data structure for the package tview table
type Package struct {
	Name         string
	Source       string
	IsInstalled  bool
	LastModified int
}
