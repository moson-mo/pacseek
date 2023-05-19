package pacseek

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/moson-mo/pacseek/internal/util"
)

// regex replacements for Gitlab URL's
// https://gitlab.archlinux.org/archlinux/devtools/-/blob/b519c8128e59e5431e2854b322e8b3a0088643cc/src/lib/api/gitlab.sh#L87
var gitlabRepl = map[string]*regexp.Regexp{
	`$1-$2`: regexp.MustCompile(`([a-zA-Z0-9]+)\+([a-zA-Z]+)`),
	`plus`:  regexp.MustCompile(`\+`),
	`-`:     regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`),
}

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
		return fmt.Sprintf(UrlRepoPkgbuild, encodePackageGitlabUrl(base))
	}
	return fmt.Sprintf(UrlAurPkgbuild, base)
}

func encodePackageGitlabUrl(pkgname string) string {
	for rep, regex := range gitlabRepl {
		pkgname = regex.ReplaceAllString(pkgname, rep)
	}
	return pkgname
}

// returns command to download and display PKGBUILD
func (ps *UI) getPkgbuildCommand(source, base string) string {
	return strings.Replace(ps.conf.ShowPkgbuildCommand, "{url}", getPkgbuildUrl(source, base), -1)
}
