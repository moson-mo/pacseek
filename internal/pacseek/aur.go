package pacseek

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

// calls the AUR rpc API (suggest type) and returns found packages (beginning with "term")
func searchAur(url, term string, timeout int) ([]Package, error) {
	packages := []Package{}
	client := http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
	}

	r, err := client.Get(url + "?v=5&type=suggest&arg=" + term)
	if err != nil {
		return packages, err
	}

	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return packages, err
	}
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
	return p
}
