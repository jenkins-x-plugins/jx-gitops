package variablefinders

import (
	"os"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// FindVersion finds the version name
func FindVersion(versionFile, branch, buildNumber string) (string, error) {
	version := ""
	if versionFile != "" {
		exists, err := files.FileExists(versionFile)
		if err != nil {
			return version, errors.Wrapf(err, "failed to check for file %s", versionFile)
		}
		if exists {
			data, err := os.ReadFile(versionFile)
			if err != nil {
				return version, errors.Wrapf(err, "failed to read version file %s", versionFile)
			}
			version = strings.TrimSpace(string(data))
		} else {
			log.Logger().Infof("version file %s does not exist", versionFile)
		}
	}
	if version == "" {
		version = os.Getenv("VERSION")
	}
	if version == "" {
		pullNumber := os.Getenv("PULL_NUMBER")
		if pullNumber != "" {
			return "0.0.0-PR-" + pullNumber + "-" + buildNumber + "-SNAPSHOT", nil
		}
		if strings.HasPrefix(branch, "PR-") {
			return "0.0.0-" + branch + "-" + buildNumber + "-SNAPSHOT", nil
		}
		log.Logger().Warnf("could not detect version from $VERSION or version file %s. Try supply the command option: --version", versionFile)
	}
	return version, nil
}
