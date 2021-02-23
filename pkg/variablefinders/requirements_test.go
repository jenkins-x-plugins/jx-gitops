package variablefinders_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	fakejx "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-gitops/pkg/variablefinders"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = true
)

func TestFindRequirements(t *testing.T) {
	ns := "jx"
	devGitURL := "https://github.com/myorg/myrepo.git"

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = devGitURL
	jxClient := fakejx.NewSimpleClientset(devEnv)

	testCases := []struct {
		path        string
		expectError bool
	}{
		{
			path: "chart_repo",
		},
		{
			path: "no_settings",
		},
		{
			path: "all",
		},
	}

	for _, tc := range testCases {
		name := tc.path
		dir := filepath.Join("test_data", name)

		runner := &fakerunner.FakeRunner{
			CommandRunner: func(command *cmdrunner.Command) (string, error) {
				if command.Name == "git" && len(command.Args) > 1 && command.Args[0] == "clone" {
					if command.Dir == "" {
						return "", errors.Errorf("no dir for git clone")
					}
					devGitPath := filepath.Join(dir, "dev-env")
					destDir := command.Dir
					if len(command.Args) > 2 {
						destDir = command.Args[2]
					}
					err := files.CopyDirOverwrite(devGitPath, destDir)
					if err != nil {
						return "", errors.Wrapf(err, "failed to copy %s to %s", devGitPath, command.Dir)
					}
					return "", nil
				}
				return "fake " + command.CLI(), nil
			},
		}

		g := cli.NewCLIClient("git", runner.Run)
		requirements, err := variablefinders.FindRequirements(g, jxClient, ns, dir)

		if tc.expectError {
			require.Error(t, err, "expected error for %s", name)
			t.Logf("got expected error %s for %s\n", err.Error(), name)
		} else {
			require.NoError(t, err, "should not fail for %s", name)
			require.NotNil(t, requirements, "should have got a requirements for %s", name)
		}

		expectedPath := filepath.Join(dir, "expected.yml")
		generatedFile := filepath.Join(tmpDir, name+"-requirements.yml")
		err = yamls.SaveFile(requirements, generatedFile)
		require.NoError(t, err, "failed to save generated requirements %s", generatedFile)

		if generateTestOutput {
			data, err := ioutil.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = ioutil.WriteFile(expectedPath, data, 0666)
			require.NoError(t, err, "failed to save file %s", expectedPath)
			continue
		}
		testhelpers.AssertTextFilesEqual(t, expectedPath, generatedFile, "generated requirements file for test "+name)
	}
}
