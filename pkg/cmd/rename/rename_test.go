package rename_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/rename"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameYamlFiles(t *testing.T) {
	srcFile := filepath.Join("test_data")
	require.DirExists(t, srcFile)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	err = files.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	_, o := rename.NewCmdRename()
	o.Dir = tmpDir

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcFile, tmpDir)

	t.Logf("split files in dir %s\n", tmpDir)

	expectedFiles := []string{
		"tekton-pipelines-webhook-sa.yaml",
		"pipelines.tekton.dev-crd.yaml",
		"cheese-svc.yaml",
		"cheese-ksvc.yaml",
	}

	for _, f := range expectedFiles {
		assert.FileExists(t, filepath.Join(tmpDir, f))
	}
}
