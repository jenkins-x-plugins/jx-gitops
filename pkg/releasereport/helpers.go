package releasereport

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/pkg/errors"
)

// LoadReleases loads the releases
func LoadReleases(path string, releases *[]*NamespaceReleases) error {
	err := yamls.LoadFile(path, releases)
	if err != nil {
		return errors.Wrapf(err, "failed to load %s", path)
	}
	return nil

}
