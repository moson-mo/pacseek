package pacseek

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type pacseekTestSuite struct {
	suite.Suite
}

func (suite *pacseekTestSuite) SetupSuite() {
	fmt.Println(">>> Setting up test suite")
}

func (suite *pacseekTestSuite) TearDownSuite() {
	fmt.Println(">>> Tests completed")
}

func TestRunTestSuite(t *testing.T) {
	suite.Run(t, new(pacseekTestSuite))
}

func (suite *pacseekTestSuite) TestInitPacmanDbs() {
	// ok
	h, err := initPacmanDbs("/var/lib/pacman", "/etc/pacman.conf", []string{})
	suite.NotNil(h, err)
	suite.Nil(err, err)

	// nok
	h, err = initPacmanDbs("/var/lib/pacman", "nonsense", []string{})
	suite.Nil(h)
	suite.NotNil(err)

	h, err = initPacmanDbs("nonsense", "/etc/pacman.conf", []string{})
	suite.Nil(h)
	suite.NotNil(err)
}

func (suite *pacseekTestSuite) TestSearchPacmanDbs() {
	h, err := initPacmanDbs("/var/lib/pacman", "/etc/pacman.conf", []string{})
	suite.NotNil(h, err)
	suite.Nil(err, err)

	// ok
	p, _, err := searchRepos(h, "glibc", "StartsWith", "Name", 1)
	suite.Nil(err, err)
	suite.Len(p, 1, "Number of packages != 1")
	p, _, err = searchRepos(h, "glibc", "StartsWith", "Name & Description", 1)
	suite.Nil(err, err)
	suite.Len(p, 1, "Number of packages != 1")

	// nok
	p, _, err = searchRepos(h, "nonsense_nonsense", "StartsWith", "Name", 1)
	suite.Nil(err, err)
	suite.Equal([]Package{}, p, "[]Packages not empty")

	p, _, err = searchRepos(nil, "nonsense_nonsense", "StartsWith", "Name", 1)
	suite.NotNil(err, err)
	suite.Equal([]Package{}, p, "[]Packages not empty")
}

func (suite *pacseekTestSuite) TestInfoPacmanDbs() {
	h, err := initPacmanDbs("/var/lib/pacman", "/etc/pacman.conf", []string{})
	suite.NotNil(h, err)
	suite.Nil(err, err)

	// ok
	p := infoPacman(h, false, "glibc")
	suite.Equal("", p.Error, "error not empty")
	suite.Equal(1, len(p.Results), "Results not 1")
	suite.Equal("glibc", p.Results[0].Name, "Name not glibc")

	// nok
	p = infoPacman(h, false, "nonsense_nonsense")
	suite.Equal("", p.Error, "error not empty")
	suite.Equal(0, len(p.Results), "Results not 0")
}

func (suite *pacseekTestSuite) TestIsInstalled() {
	h, err := initPacmanDbs("/var/lib/pacman", "/etc/pacman.conf", []string{})
	suite.NotNil(h, err)
	suite.Nil(err, err)

	// ok
	r := isPackageInstalled(h, "glibc")
	suite.True(r, "glibc not installed?")

	// nok
	r = isPackageInstalled(h, "nonsense_nonsense")
	suite.False(r, "nonsense_nonsense is installed?")
}

func (suite *pacseekTestSuite) TestSearchAur() {
	// ok
	p, err := searchAur("http://server.moson.rocks:10666/rpc", "yay", 5000, "StartsWith", "Name", 20)
	suite.Nil(err, err)
	suite.Greater(len(p), 0, "no results for yay")
	p, err = searchAur("http://server.moson.rocks:10666/rpc", "yay", 5000, "Contains", "Name", 20)
	suite.Nil(err, err)
	suite.Greater(len(p), 0, "no results for yay")
	p, err = searchAur("http://server.moson.rocks:10666/rpc", "yay", 5000, "Contains", "Name & Description", 20)
	suite.Nil(err, err)
	suite.Greater(len(p), 0, "no results for yay")
	p, err = searchAur("http://server.moson.rocks:10666/rpc", "yay", 5000, "StartsWith", "Name & Description", 20)
	suite.Nil(err, err)
	suite.Greater(len(p), 0, "no results for yay")

	// nok
	p, err = searchAur("http://server.moson.rocks:10666/rpcbla", "yay", 5000, "StartsWith", "Name", 20)
	suite.NotNil(err, err)
	suite.Equal([]Package{}, p, "[]Packages not empty")

	p, err = searchAur("nonsense", "yay", 5000, "StartsWith", "Name", 20)
	suite.NotNil(err, err)
	suite.Equal([]Package{}, p, "[]Packages not empty")
}

func (suite *pacseekTestSuite) TestInfoAur() {
	// ok
	p := infoAur("http://server.moson.rocks:10666/rpc", 5000, "yay")
	suite.Equal("", p.Error, "error not empty")
	suite.Greater(len(p.Results), 0, "no results for yay")

	// nok
	p = infoAur("http://server.moson.rocks:10666/rpcnonsense", 5000, "yay")
	suite.NotEqual("", p.Error, "error empty")
	suite.Equal(0, len(p.Results), "Results not empty")

	p = infoAur("nonsense", 5000, "yay")
	suite.NotEqual("", p.Error, "error empty")
	suite.Equal(0, len(p.Results), "Results not empty")
}
