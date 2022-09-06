package pacseek

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/moson-mo/pacseek/internal/util"
)

// download the PKGBUILD file
func getPkgbuildContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// composes the URL to a PKGBUILD file
func getPkgbuildUrl(source, base string) string {
	if util.SliceContains(getArchRepos(), source) {
		repo := "packages"
		if strings.Contains(source, "community") ||
			source == "multilib" {
			repo = "community"
		}
		return fmt.Sprintf(UrlRepoPkgbuild, repo, base)
	}
	return fmt.Sprintf(UrlAurPkgbuild, base)
}

// returns command to download and display PKGBUILD
func (ps *UI) getPkgbuildCommand(source, base string) string {
	return strings.Replace(ps.conf.ShowPkgbuildCommand, "{url}", getPkgbuildUrl(source, base), -1)
}
