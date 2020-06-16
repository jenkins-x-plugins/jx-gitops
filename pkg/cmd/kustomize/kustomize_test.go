package kustomize_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/kustomize"
	"github.com/jenkins-x/jx-gitops/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKustomize(t *testing.T) {
	_, ko := kustomize.NewCmdKustomize()

	expected := filepath.Join("test_data", "expected", "myapp")

	ko.SourceDir = filepath.Join("test_data", "source")
	ko.TargetDir = filepath.Join("test_data", "target")
	require.DirExists(t, ko.SourceDir)
	require.DirExists(t, ko.TargetDir)

	err := ko.Run()
	require.NoError(t, err, "failed to run")

	outDir := ko.OutputDir
	myapp := filepath.Join(outDir, "myapp")

	assert.NotEmpty(t, outDir, "no output dir")
	t.Logf("overlay files generated in %s\n", myapp)

	assert.FileExists(t, filepath.Join(myapp, "deployment.yaml"))
	assert.FileExists(t, filepath.Join(myapp, "ingress.yaml"))
	testhelpers.AssertFileNotExists(t, filepath.Join(myapp, "service.yaml"))

	testhelpers.AssertTextFilesEqual(t, filepath.Join(myapp, "ingress.yaml"), filepath.Join(expected, "ingress.yaml"), "kusomize")
	testhelpers.AssertTextFilesEqual(t, filepath.Join(myapp, "deployment.yaml"), filepath.Join(expected, "deployment.yaml"), "kusomize")
}
