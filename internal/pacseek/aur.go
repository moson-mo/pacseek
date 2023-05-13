package pacseek

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// calls the AUR rpc API (suggest type) and returns found packages (beginning with "term")
func searchAur(aurUrl, term string, timeout int, mode string, by string, maxResults int) ([]Package, error) {
	packages := []Package{}
	client := http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
	}

	t := "search"
	if by == "Name" {
		t = "search&by=name"
	}

	req, err := http.NewRequest("GET", aurUrl+"?v=5&type="+t+"&arg="+url.QueryEscape(term), nil)
	if err != nil {
		return packages, err
	}

	req.Header.Set("User-Agent", "pacseek/"+version)

	r, err := client.Do(req)
	if err != nil {
		return packages, err
	}

	defer r.Body.Close()

	var s RpcResult
	err = json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		return packages, err
	}

	if s.Error != "" {
		return packages, errors.New(s.Error)
	}

	// we need to sort our results here. The official aurweb /rpc endpoint is not ordering by name...
	sort.Slice(s.Results, func(i, j int) bool {
		return s.Results[i].Name < s.Results[j].Name
	})

	i := 0
	for _, pkg := range s.Results {
		// filter records
		if (mode == "StartsWith" && by == "Name" && strings.HasPrefix(pkg.Name, term)) ||
			(mode == "StartsWith" && by != "Name" && (strings.HasPrefix(pkg.Name, term) || strings.HasPrefix(strings.ToLower(pkg.Description), term))) ||
			mode == "Contains" {
			packages = append(packages, Package{
				Name:         pkg.Name,
				Source:       "AUR",
				LastModified: pkg.LastModified,
				Popularity:   pkg.Popularity,
			})
			if len(packages) >= maxResults {
				break
			}
			i++
		}
	}

	return packages, nil
}

// calls the AUR rpc API (info type) and returns package information
func infoAur(aurUrl string, timeout int, pkg ...string) RpcResult {
	client := http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
	}

	data := url.Values{}
	data.Add("v", "5")
	data.Add("type", "info")
	for _, p := range pkg {
		data.Add("arg[]", p)
	}

	req, err := http.NewRequest("POST", aurUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return RpcResult{Error: err.Error()}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "pacseek/"+version)

	r, err := client.Do(req)
	if err != nil {
		return RpcResult{Error: err.Error()}
	}

	defer r.Body.Close()

	var p RpcResult
	err = json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		return RpcResult{Error: err.Error()}
	}
	for i := 0; i < len(p.Results); i++ {
		p.Results[i].Source = "AUR"
	}

	return p
}

// calls the AUR rpc API (suggest type) and returns package names
func suggestAur(aurUrl, term string, timeout int) []string {
	packages := []string{}
	client := http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
	}

	req, err := http.NewRequest("GET", aurUrl+"?v=5&type=suggest&arg="+url.PathEscape(term), nil)
	if err != nil {
		return packages
	}

	req.Header.Set("User-Agent", "pacseek/"+version)

	r, err := client.Do(req)
	if err != nil {
		return packages
	}

	defer r.Body.Close()
	err = json.NewDecoder(r.Body).Decode(&packages)
	if err != nil {
		return packages
	}

	return packages
}
