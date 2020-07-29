// +build integration

package helm_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepHelmStream(t *testing.T) {
	_, o := helm.NewCmdHelmStream()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	tmpDir2, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp source dir")

	srcDir := filepath.Join(tmpDir2, "versionstream")
	fromSrcDir := filepath.Join("test_data", "versionstream")

	err = files.CopyDir(fromSrcDir, srcDir, false)
	require.NoError(t, err, "failed to copy from %s to %s", fromSrcDir, srcDir)

	t.Logf("generating charts to %s\n", tmpDir)

	o.Dir = srcDir
	o.OutDir = tmpDir
	o.BatchMode = true

	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "clone" && len(c.Args) > 0 {
				// lets really git clone but then fake out all other commands
				return cmdrunner.DefaultCommandRunner(c)
			}
			return "", nil
		},
	}
	hasHelm := HasHelmBinary(t, "helm")
	if !hasHelm {
		o.CommandRunner = runner.Run
	}

	o.Gitter = cli.NewCLIClient("", runner.Run)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	if hasHelm {
		templateDir := tmpDir
		require.DirExists(t, templateDir)

		t.Logf("generated templates to %s", templateDir)

		expectedPaths := []string{
			"banzaicloud-stable/vault-operator/crd.yaml",
			"bitnami/external-dns/serviceaccount.yaml",
			"external-secrets/kubernetes-external-secrets/deployment.yaml",
			"jenkins-x/lighthouse/webhooks-deployment.yaml",
			"stable/cert-manager/deployment.yaml",
		}
		for _, path := range expectedPaths {
			fullPath := filepath.Join(tmpDir, path)
			assert.FileExists(t, fullPath)
		}
	}
}
