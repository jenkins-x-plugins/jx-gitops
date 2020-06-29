package helm_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-gitops/pkg/fakes/fakegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepHelmTemplate(t *testing.T) {
	_, o := helm.NewCmdHelmTemplate()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	name := "mychart"
	o.HelmBinary = "helm"
	o.ReleaseName = name
	o.Chart = filepath.Join("test_data", name)
	o.OutDir = tmpDir
	o.BatchMode = true
	o.Gitter = fakegit.NewGitFakeClone()

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	templateDir := tmpDir
	require.DirExists(t, templateDir)

	t.Logf("generated templates to %s", templateDir)

	assert.FileExists(t, filepath.Join(templateDir, "deployment.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "service.yaml"))
}
