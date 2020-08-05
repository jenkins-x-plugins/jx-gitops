package resolve_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/resolve"
	"github.com/jenkins-x/jx-gitops/pkg/fakekpt"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepHelmfileResolve(t *testing.T) {
	_, o := resolve.NewCmdHelmfileResolve()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	srcDir := filepath.Join("test_data", "input")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy generated crds at %s to %s", srcDir, tmpDir)

	o.Dir = tmpDir

	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "clone" && len(c.Args) > 0 {
				// lets really git clone but then fake out all other commands
				return cmdrunner.DefaultCommandRunner(c)
			}
			t.Logf("running command %s in dir %s\n", c.CLI(), c.Dir)
			if c.Name == "kpt" {
				return fakekpt.FakeKpt(t, c, o.VersionStreamDir, tmpDir)
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
	helmState := &state.HelmState{}

	helmfileName := filepath.Join(o.Dir, "helmfile.yaml")
	err = yamls.LoadFile(helmfileName, helmState)
	require.NoError(t, err, "failed to load file %s", helmfileName)
	assert.NotEmpty(t, helmState.Releases, "no releases found in %s", helmfileName)

	// verify all the values files exist
	for _, release := range helmState.Releases {
		for _, v := range release.Values {
			text, ok := v.(string)
			if ok {
				fileName := filepath.Join(o.Dir, text)
				if assert.FileExists(t, fileName, "file should exist for release %s in file %s", release.Name, helmfileName) {
					t.Logf("file %s exists for release %s\n", fileName, release.Name)
				}
			}
		}
	}

	testhelpers.AssertTextFilesEqual(t, filepath.Join("test_data", "expected-helmfile.yaml"), filepath.Join(tmpDir, "helmfile.yaml"), "generated file")
}
