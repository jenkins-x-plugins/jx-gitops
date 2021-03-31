package kustomize_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kustomize"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKustomize(t *testing.T) {
	_, ko := kustomize.NewCmdKustomize()

	ko.SourceDir = filepath.Join("test_data", "source")
	ko.TargetDir = filepath.Join("test_data", "target")
	require.DirExists(t, ko.SourceDir)
	require.DirExists(t, ko.TargetDir)

	err := ko.Run()
	require.NoError(t, err, "failed to run")

	outDir := ko.OutputDir
	assert.NotEmpty(t, outDir, "no output dir")
	t.Logf("overlay files generated in %s\n", outDir)

	expected := filepath.Join("test_data", "expected", "godemo48")
	actual := filepath.Join(outDir, "godemo48")
	testhelpers.AssertFileNotExists(t, filepath.Join(actual, "deployment.yaml"))
	testhelpers.AssertFileNotExists(t, filepath.Join(actual, "service.yaml"))

	actual = filepath.Join(outDir, "myapp")
	expected = filepath.Join("test_data", "expected", "myapp")
	assert.FileExists(t, filepath.Join(actual, "deployment.yaml"))
	assert.FileExists(t, filepath.Join(actual, "ingress.yaml"))
	testhelpers.AssertFileNotExists(t, filepath.Join(actual, "service.yaml"))

	testhelpers.AssertTextFilesEqual(t, filepath.Join(actual, "ingress.yaml"), filepath.Join(expected, "ingress.yaml"), "kusomize")
	testhelpers.AssertTextFilesEqual(t, filepath.Join(actual, "deployment.yaml"), filepath.Join(expected, "deployment.yaml"), "kusomize")
}
