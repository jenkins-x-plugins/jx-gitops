package fakekpt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/pkg/errors"
)

// FakeKpt a simple command to simulate kpt using local files for use in a fake cmdrunner
func FakeKpt(t *testing.T, c *cmdrunner.Command, versionStreamDir string, targetDir string) (string, error) {
	if len(c.Args) < 4 {
		return "", errors.Errorf("unsupported kpt command %s", c.CLI())
	}

	valuesDir := c.Args[3]

	// lets trim the versionStream folder from the valuesDir
	dirs := strings.Split(valuesDir, string(os.PathSeparator))
	srcValuesDir := filepath.Join(versionStreamDir, filepath.Join(dirs[1:]...))

	// lets copy the file from the src dir to the target to simulate kpt
	targetValuesDir := filepath.Join(targetDir, valuesDir)
	t.Logf("copying version stream dir %s to %s\n", srcValuesDir, targetValuesDir)

	err := files.CopyDir(srcValuesDir, targetValuesDir, true)
	if err != nil {
		return "", errors.Wrapf(err, "failed to copy %s to %s", srcValuesDir, targetValuesDir)
	}
	return "", nil
}
