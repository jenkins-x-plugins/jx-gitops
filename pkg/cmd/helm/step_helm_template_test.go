package helm_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepHelmTemplate(t *testing.T) {
	_, o := helm.NewCmdHelmTemplate()

	helmBin := "helm"
	hasHelm := HasHelmBinary(t, helmBin)
	if !hasHelm {
		return
	}

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	name := "mychart"
	o.HelmBinary = helmBin
	o.ReleaseName = name
	o.Chart = filepath.Join("test_data", name)
	o.OutDir = tmpDir
	o.BatchMode = true

	runner := &fakerunner.FakeRunner{}
	o.Gitter = cli.NewCLIClient("", runner.Run)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	if hasHelm {
		templateDir := tmpDir
		require.DirExists(t, templateDir)

		t.Logf("generated templates to %s", templateDir)

		assert.FileExists(t, filepath.Join(templateDir, "deployment.yaml"))
		assert.FileExists(t, filepath.Join(templateDir, "service.yaml"))
	}
}

// HasHelmBinary lets test if we are running the tests in a container with the helm binary
func HasHelmBinary(t *testing.T, helmBin string) bool {
	c := &cmdrunner.Command{
		Name: helmBin,
		Args: []string{"version", "--short"},
	}
	version, err := cmdrunner.DefaultCommandRunner(c)
	if err != nil {
		t.Logf("could not find the %s binary so lets disable helm tests for this image %s", helmBin, err.Error())
		return false
	}
	t.Logf("found helm version %s", version)
	return true
}
