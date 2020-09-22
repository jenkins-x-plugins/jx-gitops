package variablefinders

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/pkg/errors"
)

//  FindVersion finds the version name
func FindVersion(versionFile, branch, buildNumber string) (string, error) {
	version := ""
	if versionFile != "" {
		exists, err := files.FileExists(versionFile)
		if err != nil {
			return version, errors.Wrapf(err, "failed to check for file %s", versionFile)
		}
		if exists {
			data, err := ioutil.ReadFile(versionFile)
			if err != nil {
				return version, errors.Wrapf(err, "failed to read version file %s", versionFile)
			}
			version = strings.TrimSpace(string(data))
		}
	}
	if version == "" {
		version = os.Getenv("VERSION")
	}
	if version == "" {
		if strings.HasPrefix(branch, "PR-") {
			return "0.0.0-SNAPSHOT-" + branch + "-" + buildNumber, nil
		}
		return version, errors.Errorf("could not detect version from $VERSION or version file %s. Try supply the command option: --version", versionFile)
	}
	return version, nil
}
