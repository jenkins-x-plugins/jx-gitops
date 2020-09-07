package helmhelpers

import (
	"net/url"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
	"k8s.io/client-go/rest"
)

// AddHelmRepositories ensures the repositories in the helmfile are added to helm
// so that we can use helm templating etc
func AddHelmRepositories(helmState state.HelmState, runner cmdrunner.CommandRunner, ignoreRepositories []string) error {
	repoMap := map[string]string{
		"jx": "http://chartmuseum.jenkins-x.io",
	}
	for _, repo := range helmState.Repositories {
		repoMap[repo.Name] = repo.URL
	}

	for repoName, repoURL := range repoMap {
		if stringhelpers.StringArrayIndex(ignoreRepositories, repoURL) >= 0 {
			continue
		}
		c := &cmdrunner.Command{
			Name: "helm",
			Args: []string{"repo", "add", repoName, repoURL},
		}
		err := RunCommandAndLogOutput(runner, c, nil, []string{" has been added to your repositories"})
		if err != nil {
			return errors.Wrap(err, "failed to add helm repo")
		}
		log.Logger().Debugf("added helm repository %s %s", repoName, repoURL)
	}
	return nil
}

// IsWhitespaceOrComments returns true if the text is empty, whitespace or comments only
func IsWhitespaceOrComments(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "--") {
			return false
		}
	}
	return true
}

// RunCommandAndLogOutput runs the command and outputs info or debug level logging
func RunCommandAndLogOutput(commandRunner cmdrunner.CommandRunner, c *cmdrunner.Command, debugPrefixes []string, infoPrefixes []string) error {
	if commandRunner == nil {
		commandRunner = cmdrunner.QuietCommandRunner
	}
	text, err := commandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run %s", c.CLI())
	}

	lines := strings.Split(text, "\n")
	lastLineDebug := false
	for _, line := range lines {
		if stringhelpers.HasPrefix(line, debugPrefixes...) || stringhelpers.HasSuffix(line, infoPrefixes...) {
			lastLineDebug = true
		} else if strings.TrimSpace(line) != "" {
			lastLineDebug = false
		}
		if lastLineDebug {
			log.Logger().Debug(line)
		} else {
			log.Logger().Info(line)
		}
	}
	return nil
}

// FindClusterLocalRepositories finds any cluster local repositories such as http://bucketrepo/bucketrepo/charts/
func FindClusterLocalRepositories(repos []state.RepositorySpec) ([]string, error) {
	var answer []string
	for _, repo := range repos {
		if repo.URL == "" {
			continue
		}
		u, err := url.Parse(repo.URL)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse repository URL %s", repo.URL)
		}
		h := u.Host

		// if we have no sub domain assume local
		if !strings.Contains(h, ".") {
			answer = append(answer, repo.URL)
		}
	}
	return answer, nil
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
