package add_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/add"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/fakekpt"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles/testhelmfile"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/stretchr/testify/require"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestStepHelmfileAdd(t *testing.T) {
	testCases := []struct {
		chart       string
		repository  string
		namespace   string
		version     string
		releaseName string
	}{
		{
			chart:       "ingress-nginx/ingress-nginx",
			namespace:   "nginx",
			version:     "4.0",
			releaseName: "nginx-ingress",
		},
		{
			chart:       "jenkins-x/new-cheese",
			namespace:   "cheese",
			releaseName: "mythingy",
		},
		{
			chart: "jenkins-x/jx-test-collector",
		},
		{
			chart:      "jenkins/jenkins-operator",
			repository: "https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart",
		},
	}

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	srcDir := filepath.Join("test_data", "input")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy generated data at %s to %s", srcDir, tmpDir)

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

	for _, tc := range testCases {
		_, o := add.NewCmdHelmfileAdd()
		o.Dir = tmpDir
		o.Chart = tc.chart
		if tc.namespace == "" {
			tc.namespace = "jx"
		}
		o.Namespace = tc.namespace
		o.ReleaseName = tc.releaseName
		o.Repository = tc.repository
		o.Version = tc.version

		t.Logf("installing chart %s\n", o.Chart)

		o.CommandRunner = runner.Run
		o.Gitter = cli.NewCLIClient("", runner.Run)

		err = o.Run()
		require.NoError(t, err, "failed to run the command")
	}

	t.Logf("generated files to %s\n", tmpDir)

	testhelmfile.AssertHelmfiles(t, filepath.Join("test_data", "expected"), tmpDir, generateTestOutput)
}
