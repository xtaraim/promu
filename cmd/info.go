// Copyright © 2016 Prometheus Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Print info about current project and exit",
	Long:  `Print info about current project and exit`,
	Run: func(cmd *cobra.Command, args []string) {
		runInfo()
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(infoCmd)
}

// ProjectInfo represents current project useful informations.
type ProjectInfo struct {
	Branch   string
	Name     string
	Owner    string
	Repo     string
	Revision string
	Version  string
}

// NewProjectInfo returns a new ProjectInfo.
func NewProjectInfo() (ProjectInfo, error) {
	projectInfo := ProjectInfo{}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		repo, err := os.Getwd()
		if err != nil {
			return projectInfo, errors.Wrapf(err, "Couldn't get current working directory")
		}
		repo = strings.TrimPrefix(repo, os.Getenv("GOPATH"))
		repo = strings.TrimPrefix(repo, "/src/")

		user, err := user.Current()
		if err != nil {
			return projectInfo, errors.Wrapf(err, "Couldn't get current user")
		}

		projectInfo = ProjectInfo{
			Branch:   "non-git",
			Name:     filepath.Base(repo),
			Owner:    user.Username,
			Repo:     repo,
			Revision: "non-git",
		}
	} else {
		repo, err := repoLocation()
		if err != nil {
			return projectInfo, errors.Wrapf(err, "Couldn't parse repo location")
		}
		projectInfo = ProjectInfo{
			Branch:   shellOutput("git rev-parse --abbrev-ref HEAD"),
			Name:     filepath.Base(repo),
			Owner:    filepath.Base(filepath.Dir(repo)),
			Repo:     repo,
			Revision: shellOutput("git rev-parse HEAD"),
		}
	}

	version, err := findVersion()
	if err != nil {
		warn(errors.Wrap(err, "Unable to find project's version"))
	} else {
		projectInfo.Version = version
	}

	return projectInfo, nil
}

func runInfo() {
	fmt.Println("Name:", info.Name)
	fmt.Println("Version:", info.Version)
	fmt.Println("Owner:", info.Owner)
	fmt.Println("Repo:", info.Repo)
	fmt.Println("Branch:", info.Branch)
	fmt.Println("Revision:", info.Revision)
}

// Convert SCP-like URL to SSH URL(e.g. [user@]host.xz:path/to/repo.git/)
// ref. http://git-scm.com/docs/git-fetch#_git_urls
// (golang hasn't supported Perl-like negative look-behind match)
var hasSchemePattern = regexp.MustCompile("^[^:]+://")
var scpLikeUrlPattern = regexp.MustCompile("^([^@]+@)?([^:]+):/?(.+)$")

func repoLocation() (string, error) {
	repo := shellOutput("git config --get remote.origin.url")

	if !hasSchemePattern.MatchString(repo) && scpLikeUrlPattern.MatchString(repo) {
		matched := scpLikeUrlPattern.FindStringSubmatch(repo)
		user := matched[1]
		host := matched[2]
		path := matched[3]
		repo = fmt.Sprintf("ssh://%s%s/%s", user, host, path)
	}

	u, err := url.Parse(repo)
	if err != nil {
		return "", err
	}

	repo = fmt.Sprintf("%s%s", strings.Split(u.Host, ":")[0], u.Path)
	repo = strings.TrimSuffix(repo, ".git")
	return repo, nil
}

func findVersion() (string, error) {
	var files = []string{"VERSION", "version/VERSION"}
	for _, file := range files {
		if fileExists(file) {
			return readFile(file), nil
		}
	}
	return "", errors.New("missing `VERSION` or `version/VERSION` file")
}
