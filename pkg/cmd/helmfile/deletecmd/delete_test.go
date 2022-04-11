package deletecmd_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/deletecmd"
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

func TestStepHelmfileDelete(t *testing.T) {
	testCases := []struct {
		chart     string
		namespace string
		dir       string
	}{
		{
			chart: "ingress-nginx",
			dir:   "local",
		},
		{
			chart: "ingress-nginx/ingress-nginx",
			dir:   "fullname",
		},
		{
			chart: "does-not-exist",
			dir:   "nochange",
		},
		{
			chart: "cheese",
			dir:   "all",
		},
		{
			chart:     "cheese",
			namespace: "jx-production",
			dir:       "prod",
		},
	}

	tmpDir := t.TempDir()

	t.Logf("generating files to %s\n", tmpDir)

	srcDir := filepath.Join("test_data")
	require.DirExists(t, srcDir)

	err := files.CopyDirOverwrite(srcDir, tmpDir)
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
		_, o := deletecmd.NewCmdHelmfileDelete()
		o.Dir = filepath.Join(tmpDir, tc.dir, "input")
		o.Details.Chart = tc.chart
		o.Details.Namespace = tc.namespace

		t.Logf("deleting chart %s\n", o.Details.Chart)

		o.CommandRunner = runner.Run
		o.Gitter = cli.NewCLIClient("", runner.Run)

		err = o.Run()
		require.NoError(t, err, "failed to run the command")

		expectedDir := filepath.Join("test_data", tc.dir, "expected")
		testhelmfile.AssertHelmfiles(t, expectedDir, o.Dir, generateTestOutput)
	}
}
