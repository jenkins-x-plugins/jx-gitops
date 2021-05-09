package structure_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/structure"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelmfileStructure(t *testing.T) {

	srcDir := "test_data"
	require.DirExists(t, srcDir)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy test files at %s to %s", srcDir, tmpDir)

	o := structure.Options{
		Dir: tmpDir,
	}
	err = o.Run()

	assert.NoError(t, err)

	assert.FileExists(t, filepath.Join(tmpDir, "helmfiles", "jx", "helmfile.yaml"), "expected generated file")
	assert.FileExists(t, filepath.Join(tmpDir, "helmfiles", "tekton-pipelines", "helmfile.yaml"), "expected generated file")

	testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected-helmfile.yaml"), filepath.Join(tmpDir, "helmfile.yaml"), "generated file: helmfile.yaml")
	jxFolder := filepath.Join(tmpDir, "helmfiles", "jx")
	tektonFolder := filepath.Join(tmpDir, "helmfiles", "tekton-pipelines")
	testhelpers.AssertTextFilesEqual(t, filepath.Join(jxFolder, "expected-helmfile.yaml"), filepath.Join(jxFolder, "helmfile.yaml"), "generated file: helmfile.yaml")
	testhelpers.AssertTextFilesEqual(t, filepath.Join(tektonFolder, "expected-helmfile.yaml"), filepath.Join(tektonFolder, "helmfile.yaml"), "generated file: helmfile.yaml")

}
