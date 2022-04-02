package pacseek

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// calls the AUR rpc API (suggest type) and returns found packages (beginning with "term")
func searchAur(url, term string, timeout int, mode string, by string, maxResults int) ([]Package, error) {
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

	r, err := client.Get(url + "?v=5&type=" + t + "&arg=" + term)
	if err != nil {
		return packages, err
	}

	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
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
		i := 0
		for _, pkg := range s.Results {
			if mode == "StartsWith" && (strings.HasPrefix(pkg.Name, term) || strings.HasPrefix(pkg.Description, term)) {
				packages = append(packages, Package{
					Name:   pkg.Name,
					Source: "AUR",
				})
			} else if mode == "Contains" {
				packages = append(packages, Package{
					Name:   pkg.Name,
					Source: "AUR",
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
func infoAur(url, pkg string, timeout int) RpcResult {
	client := http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
	}
	r, err := client.Get(url + "?v=5&type=info&arg=" + pkg)
	if err != nil {
		return RpcResult{Error: err.Error()}
	}

	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
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
