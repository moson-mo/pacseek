package pacseek

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/moson-mo/pacseek/internal/util"
)

type RegexReplace struct {
	repl  string
	match *regexp.Regexp
}

// regex replacements for Gitlab URL's
// https://gitlab.archlinux.org/archlinux/devtools/-/blob/6ce666a1669235749c17d5c44d8a24dea4a135da/src/lib/api/gitlab.sh#L95
var gitlabRepl = []RegexReplace{
	{repl: `$1-$2`, match: regexp.MustCompile(`([a-zA-Z0-9]+)\+([a-zA-Z]+)`)},
	{repl: `plus`, match: regexp.MustCompile(`\+`)},
	{repl: `-`, match: regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)},
	{repl: `-`, match: regexp.MustCompile(`[_\-]{2,}`)},
	{repl: `unix-tree`, match: regexp.MustCompile(`^tree$`)},
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
	for _, regex := range gitlabRepl {
		pkgname = regex.match.ReplaceAllString(pkgname, regex.repl)
	}
	return pkgname
}

// returns command to download and display PKGBUILD
func (ps *UI) getPkgbuildCommand(source, base string) string {
	return strings.Replace(ps.conf.ShowPkgbuildCommand, "{url}", getPkgbuildUrl(source, base), -1)
}
