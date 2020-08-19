package versionstreamer

import (
	"github.com/jenkins-x/jx-helpers/pkg/versionstream"
	"github.com/pkg/errors"
)

// MatchRepositoryPrefix returns the prefix match if it can be found from the version stream
func MatchRepositoryPrefix(prefixes *versionstream.RepositoryPrefixes, prefix string) (string, error) {
	if prefixes == nil {
		return "", errors.Errorf("no repository prefixes found in version stream")
	}
	// default to first URL
	repoURL := prefixes.URLsForPrefix(prefix)

	if len(repoURL) == 0 {
		return "", errors.Errorf("no matching repository for for prefix %s", prefix)
	}
	return repoURL[0], nil
}
