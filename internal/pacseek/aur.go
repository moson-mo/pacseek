package pacseek

import (
	"encoding/json"
	"errors"
	"io"
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
	t := "suggest"
	if mode == "Contains" {
		t = "search&by=name"
	}
	if by == "Name & Description" {
		t = "search"
	}

	req, err := http.NewRequest("GET", aurUrl+"?v=5&type="+t+"&arg="+term, nil)
	if err != nil {
		return packages, err
	}

	req.Header.Set("User-Agent", "pacseek/"+version)

	r, err := client.Do(req)
	if err != nil {
		return packages, err
	}

	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return packages, err
	}

	if t == "suggest" {
		var s []string
		err = json.Unmarshal(b, &s)
		if err != nil {
			return packages, err
		}
		for _, pkg := range s {
			packages = append(packages, Package{
				Name:   pkg,
				Source: "AUR",
			})
		}
	} else {
		var s RpcResult
		err = json.Unmarshal(b, &s)
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
			if (mode == "StartsWith" && (strings.HasPrefix(pkg.Name, term) || strings.HasPrefix(pkg.Description, term))) ||
				mode == "Contains" {
				packages = append(packages, Package{
					Name:         pkg.Name,
					Source:       "AUR",
					LastModified: pkg.LastModified,
					Popularity:   pkg.Popularity,
				})
			}
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
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return RpcResult{Error: err.Error()}
	}
	var p RpcResult
	err = json.Unmarshal(b, &p)
	if err != nil {
		return RpcResult{Error: err.Error()}
	}
	for i := 0; i < len(p.Results); i++ {
		p.Results[i].Source = "AUR"
	}

	return p
}
