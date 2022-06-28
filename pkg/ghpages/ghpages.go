package ghpages

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// CloneGitHubPagesToDir clones the github pages repository to a directory
func CloneGitHubPagesToDir(g gitclient.Interface, repoURL, branch, username, password string) (string, error) {
	dir, err := ioutil.TempDir("", "gh-pages-tmp-")
	if err != nil {
		return dir, errors.Wrapf(err, "failed to create temp dir")
	}

	gitCloneURL, err := GitHubPagesCloneURL(repoURL, username, password)
	if err != nil {
		return dir, errors.Wrapf(err, "failed to get github pages clone URL")
	}

	_, err = g.Command(dir, "clone", gitCloneURL, "--branch", branch, "--single-branch", dir)
	if err != nil {
		log.Logger().Infof("assuming the remote branch does not exist so lets create it")

		_, err = gitclient.CloneToDir(g, gitCloneURL, dir)
		if err != nil {
			return dir, errors.Wrapf(err, "failed to clone repository %s to directory: %s", gitCloneURL, dir)
		}

		// now lets create an empty orphan branch: see https://stackoverflow.com/a/13969482/2068211
		_, err = g.Command(dir, "checkout", "--orphan", branch)
		if err != nil {
			return dir, errors.Wrapf(err, "failed to checkout an orphan branch %s in dir %s", branch, dir)
		}

		_, err = g.Command(dir, "rm", "--cached", "-r", ".")
		if err != nil {
			return dir, errors.Wrapf(err, "failed to remove the cached git files in dir %s", dir)
		}

		// lets remove all the files other than .git
		files, err := os.ReadDir(dir)
		if err != nil {
			return dir, errors.Wrapf(err, "failed to read files in dir %s", dir)
		}
		for _, f := range files {
			name := f.Name()
			if name == ".git" {
				continue
			}
			path := filepath.Join(dir, name)
			err = os.RemoveAll(path)
			if err != nil {
				return dir, errors.Wrapf(err, "failed to remove path %s", path)
			}
		}
	}
	if err == nil {
		_, err = g.Command(dir, "remote", "set-url", "origin", gitCloneURL)
		if err != nil {
			return dir, errors.Wrapf(err, "failed to set origin URL")
		}
	}
	return dir, nil
}

// GitHubPagesCloneURL adds the optional username and password to the github pages URL for cloning
func GitHubPagesCloneURL(repoURL, username, password string) (string, error) {
	if username == "" || password == "" {
		return repoURL, nil
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse git URL %s", repoURL)
	}
	u.User = url.UserPassword(username, password)
	return u.String(), nil
}
