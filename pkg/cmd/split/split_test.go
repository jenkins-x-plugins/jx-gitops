package split_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x/jx-gitops/pkg/testhelpers"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitYamlFiles(t *testing.T) {
	srcFile := filepath.Join("test_data")
	require.DirExists(t, srcFile)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	err = util.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	o := &split.Options{
		Dir: tmpDir,
	}

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcFile, tmpDir)

	t.Logf("split files in dir %s\n", tmpDir)

	assert.FileExists(t, filepath.Join(tmpDir, "two", "foo-svc.yaml"))
	assert.FileExists(t, filepath.Join(tmpDir, "two", "foo-svc2.yaml"))
	assert.FileExists(t, filepath.Join(tmpDir, "no-separator", "cheese-svc.yaml"))

	assert.FileExists(t, filepath.Join(tmpDir, "separator", "cheese-svc.yaml"))
	testhelpers.AssertFileNotExists(t, filepath.Join(tmpDir, "separator", "cheese-svc2.yaml"))

	assert.FileExists(t, filepath.Join(tmpDir, "separator-in-crd", "crd.yaml"))
	testhelpers.AssertFileNotExists(t, filepath.Join(tmpDir, "separator-in-crd", "crd2.yaml"))

	assert.FileExists(t, filepath.Join(tmpDir, "comment", "foo-svc.yaml"))
	testhelpers.AssertFileNotExists(t, filepath.Join(tmpDir, "comment", "foo-svc2.yaml"))
}

func TestSplitHelmTemplateYamlFiles(t *testing.T) {
	srcFile := filepath.Join("test_data", "helm")
	require.DirExists(t, srcFile)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	err = util.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	o := &split.Options{
		Dir: tmpDir,
	}

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcFile, tmpDir)

	t.Logf("split files in dir %s\n", tmpDir)

	assert.FileExists(t, filepath.Join(tmpDir, "tekton.yaml"))
	i := 2
	for i != 30 {
		assert.FileExists(t, filepath.Join(tmpDir, fmt.Sprintf("tekton%d.yaml", i)))
		i++
	}
}
