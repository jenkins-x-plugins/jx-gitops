package resolve_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-apps/pkg/jxapps"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/apps/resolve"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepJxAppsResolve(t *testing.T) {
	_, o := resolve.NewCmdJxAppsResolve()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	srcDir := filepath.Join("test_data", "input")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy generated crds at %s to %s", srcDir, tmpDir)

	o.Dir = tmpDir
	o.VersionStreamDir = filepath.Join("test_data", "versionstream")

	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "clone" && len(c.Args) > 0 {
				// lets really git clone but then fake out all other commands
				return cmdrunner.DefaultCommandRunner(c)
			}
			t.Logf("running command %s in dir %s\n", c.CLI(), c.Dir)
			if c.Name == "kpt" && len(c.Args) > 3 {
				valuesDir := c.Args[3]

				// lets trim the versionStream folder from the valuesDir
				dirs := strings.Split(valuesDir, string(os.PathSeparator))
				srcValuesDir := filepath.Join(o.VersionStreamDir, filepath.Join(dirs[1:]...))

				// lets copy the file from the src dir to the target to simulate kpt
				targetValuesDir := filepath.Join(tmpDir, valuesDir)
				t.Logf("copying version stream dir %s to %s\n", srcValuesDir, targetValuesDir)

				err = files.CopyDirOverwrite(srcValuesDir, targetValuesDir)
				if err != nil {
					return "", errors.Wrapf(err, "failed to copy %s to %s", srcValuesDir, targetValuesDir)
				}
			}
			return "", nil
		},
	}
	o.CommandRunner = runner.Run
	o.Gitter = cli.NewCLIClient("", runner.Run)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	t.Logf("generated files to %s\n", o.Dir)

	// lets assert that all the values files exist
	appCfg, appCfgFile, err := jxapps.LoadAppConfig(o.Dir)
	require.NoError(t, err, "failed to load apps in dir %s", o.Dir)
	assert.NotEmpty(t, appCfg.Apps, "no apps found in %s", appCfgFile)

	// verify all the values files exist
	for _, app := range appCfg.Apps {
		for _, v := range app.Values {
			fileName := filepath.Join(o.Dir, v)
			if assert.FileExists(t, fileName, "file should exist for app %s in file %s", app.Name, appCfgFile) {
				t.Logf("file %s exists for app %s\n", fileName, app.Name)
			}
		}
	}

	testhelpers.AssertTextFilesEqual(t, filepath.Join("test_data", "expected-jx-apps.yml"), filepath.Join(tmpDir, "jx-apps.yml"), "generated file")
}
