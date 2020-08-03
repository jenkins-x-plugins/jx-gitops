package merge_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/merge"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestRequirementsMerge(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcFile := filepath.Join("test_data")
	require.DirExists(t, srcFile)

	err = files.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	// now lets run the command
	_, o := merge.NewCmdRequirementsMerge()
	o.Dir = tmpDir
	o.File = filepath.Join(tmpDir, "changes.yml")

	t.Logf("merging requirements in dir %s\n", tmpDir)

	err = o.Run()
	require.NoError(t, err, "failed to run git setup")

	testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected.yml"), filepath.Join(tmpDir, config.RequirementsConfigFileName), "merged file")
}
