package resolve_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-api/v3/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/resolve"
	"github.com/jenkins-x/jx-gitops/pkg/fakekpt"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestStepHelmfileResolve(t *testing.T) {
	fileNames, err := ioutil.ReadDir("test_data")
	assert.NoError(t, err)

	// lets find the helm binary on the $PATH or download a plugin if inside CI/CD
	helmBin := "helm"
	c := &cmdrunner.Command{
		Name: "helm",
		Args: []string{"version"},
	}
	_, err = cmdrunner.DefaultCommandRunner(c)
	if err != nil {
		t.Logf("failed to run %s so downloading the helm binary\n", c.CLI())

		helmBin, err = plugins.GetHelmBinary("")
		require.NoError(t, err, "failed to download helm binary")
		require.NotEmpty(t, helmBin, "could not find helm plugin")
	}

	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()

			t.Logf("running test %s\n", name)

			_, o := resolve.NewCmdHelmfileResolve()

			tmpDir, err := ioutil.TempDir("", "")
			require.NoError(t, err, "failed to create tmp dir")

			srcDir := filepath.Join("test_data", name)
			require.DirExists(t, srcDir)

			err = files.CopyDirOverwrite(srcDir, tmpDir)
			require.NoError(t, err, "failed to copy generated crds at %s to %s", srcDir, tmpDir)

			o.Dir = tmpDir
			o.HelmBinary = helmBin
			o.TestOutOfCluster = true

			runner := &fakerunner.FakeRunner{
				CommandRunner: func(c *cmdrunner.Command) (string, error) {
					if c.Name == "clone" && len(c.Args) > 0 {
						// lets really git clone but then fake out all other commands
						return cmdrunner.DefaultCommandRunner(c)
					}
					t.Logf("running command %s in dir %s\n", c.CLI(), c.Dir)
					if c.Name == "kpt" {
						return fakekpt.FakeKpt(t, c, filepath.Join("test_data", "input", "versionStream"), tmpDir)
					}
					return "", nil
				},
			}
			o.CommandRunner = runner.Run
			o.Gitter = cli.NewCLIClient("", runner.Run)
			o.UpdateMode = true
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

			testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected-helmfile.yaml"), filepath.Join(tmpDir, "helmfile.yaml"), "generated file: "+name)

			// lets assert that we don't add the bucket repo if we are not in a cluster
			if !IsInCluster() {
				for _, cmd := range runner.OrderedCommands {
					if cmd.Name == "helm" {
						assert.NotEqual(t, []string{"repo", "add", "dev", "http://bucketrepo/bucketrepo/charts/"}, cmd.Args, "should not have added a cluster local repository for %s", name)
					}
				}
			}

			requirements, _, err := config.LoadRequirementsConfig(o.Dir, false)
			require.NoError(t, err, "failed to load requirements file from dir %s", o.Dir)
			assert.Equal(t, "https://github.com/jenkins-x/jx3-pipeline-catalog.git", requirements.BuildPacks.BuildPackLibrary.GitURL, "requirements.BuildPacks.BuildPackLibrary.GitURL")

			for _, c := range runner.OrderedCommands {
				t.Logf("fake command: %s\n", c.CLI())
			}

			require.FileExists(t, filepath.Join(o.Dir, ".jx", "git-operator", "filename.txt"), "should have generated the git operator job file name")
		}
	}
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
