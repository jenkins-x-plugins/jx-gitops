package merge

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// FetchAndMergeSHAs merges any SHAs into the baseBranch which has a tip of baseSha,
// fetching the commits from remote for the git repo in dir. It will try to fetch individual commits (
// if the remote repo supports it - see https://github.
// com/git/git/commit/68ee628932c2196742b77d2961c5e16360734a62) otherwise it uses git remote update to pull down the
// whole repo.
func FetchAndMergeSHAs(gitter gitclient.Interface, SHAs []string, baseBranch string, baseSha string, remote string, dir string) error {
	log.Logger().Infof("using base branch %s and base sha %s", info(baseBranch), info(baseSha))

	refspecs := make([]string, 0)
	for _, sha := range SHAs {
		refspecs = append(refspecs, fmt.Sprintf("%s:", sha))
	}
	refspecs = append(refspecs, fmt.Sprintf("%s:", baseSha))

	args := append([]string{"fetch", remote}, refspecs...)

	_, err := gitter.Command(dir, args...)
	if err != nil {
		log.Logger().Warnf("failed to run git %s got %s", strings.Join(args, " "), err.Error())
		//return errors.Wrapf(err, "failed to fetch shas")
		/*
			// Unshallow fetch failed, so do a full unshallow
			// First ensure we actually have the branch refs
			args := append([]string{"fetch", remote}, refspecs...)
			_, err = gitter.Command(dir, args...)
			if err != nil {

				err = gitter.RemoteUpdate(dir)
				if err != nil {
					return errors.Wrapf(err, "updating remote %s", remote)
				}
				log.Logger().Debugf("ran %s in %s", util.ColorInfo("git remote update"), dir)
			}
			log.Logger().Debugf("ran git fetch %s %s in %s", remote, strings.Join(refspecs, " "), dir)

			err = Unshallow(dir, gitter)
			if err != nil {
				return errors.WithStack(err)
			}
			log.Logger().Debugf("Unshallowed git repo in %s", dir)
		*/
	}

	branches, err := LocalBranches(gitter, dir)
	if err != nil {
		return errors.Wrapf(err, "listing local branches")
	}
	found := false
	for _, b := range branches {
		if b == baseBranch {
			found = true
			break
		}
	}
	if !found {
		_, err = gitter.Command(dir, "branch", baseBranch)
		if err != nil {
			return errors.Wrapf(err, "creating branch %s", baseBranch)
		}
	}
	// Ensure we are on baseBranch
	err = gitclient.Checkout(gitter, dir, baseBranch)
	if err != nil {
		return errors.Wrapf(err, "checking out %s", baseBranch)
	}
	log.Logger().Debugf("ran git checkout %s in %s", baseBranch, dir)
	// Ensure we are on the right revision
	_, err = gitter.Command(dir, "reset", "--hard", baseBranch)
	if err != nil {
		return errors.Wrapf(err, "resetting %s to %s", baseBranch, baseSha)
	}
	log.Logger().Debugf("ran git reset --hard %s in %s", baseSha, dir)
	_, err = gitter.Command(dir, "clean", "-fd", ".")
	if err != nil {
		return errors.Wrapf(err, "cleaning up the git repo")
	}
	log.Logger().Debugf("ran clean -fd . in %s", dir)
	// Now do the merges
	for _, sha := range SHAs {
		log.Logger().Infof("merging sha: %s", info(sha))
		_, err = gitter.Command(dir, "merge", sha)
		if err != nil {
			return errors.Wrapf(err, "merging %s into master", sha)
		}
	}
	return nil
}

// LocalBranches will list all local branches
func LocalBranches(g gitclient.Interface, dir string) ([]string, error) {
	text, err := g.Command(dir, "branch")
	answer := make([]string, 0)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		columns := strings.Split(line, " ")
		for _, col := range columns {
			if col != "" && col != "*" {
				answer = append(answer, col)
				break
			}
		}
	}
	return answer, nil
}
