package resolve_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

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
	tests := []struct {
		folder     string
		namespaces []string
	}{
		{
			folder:     "no-versionstream",
			namespaces: []string{"jx", "external-secrets", "foo", "tekton-pipelines"},
		},
		{
			folder:     "bucketrepo-svc",
			namespaces: []string{"jx", "tekton-pipelines"},
		},
		{
			folder:     "local-secrets",
			namespaces: []string{"jx", "secret-infra", "tekton-pipelines"},
		},
		{
			folder:     "input",
			namespaces: []string{"foo", "jx", "secret-infra", "tekton-pipelines"},
		},
	}

	// lets find the helm binary on the $PATH or download a plugin if inside CI/CD
	helmBin := "helm"
	c := &cmdrunner.Command{
		Name: "helm",
		Args: []string{"version"},
	}
	_, err := cmdrunner.DefaultCommandRunner(c)
	if err != nil {
		t.Logf("failed to run %s so downloading the helm binary\n", c.CLI())

		helmBin, err = plugins.GetHelmBinary("")
		require.NoError(t, err, "failed to download helm binary")
		require.NotEmpty(t, helmBin, "could not find helm plugin")
	}

	for _, test := range tests {

		name := test.folder

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
		assert.Empty(t, helmState.Releases, "releases found in %s and they should be in nested states", helmfileName)

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

		for _, ns := range test.namespaces {
			expectedHelmfile := fmt.Sprintf("expected-%s-helmfile.yaml", ns)
			testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, expectedHelmfile), filepath.Join(tmpDir, "helmfiles", ns, "helmfile.yaml"), "generated file: "+name)

		}

		// lets assert that we don't add the bucket repo if we are not in a cluster
		if !IsInCluster() {
			for _, cmd := range runner.OrderedCommands {
				if cmd.Name == "helm" {
					assert.NotEqual(t, []string{"repo", "add", "dev", "http://bucketrepo/bucketrepo/charts/"}, cmd.Args, "should not have added a cluster local repository for %s", name)
				}
			}
		}

		for _, c := range runner.OrderedCommands {
			t.Logf("fake command: %s\n", c.CLI())
		}

		require.FileExists(t, filepath.Join(o.Dir, ".jx", "git-operator", "filename.txt"), "should have generated the git operator job file name")

		if name == "input" {
			require.FileExists(t, filepath.Join(o.Dir, "jx-global-values.yaml"), "should have renamed imagePullSecrets.yaml")
		}
	}
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
