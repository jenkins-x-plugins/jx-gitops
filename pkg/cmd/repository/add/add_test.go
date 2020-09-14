package add_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/repository/add"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestRepositoryAdd(t *testing.T) {
	sourceData := filepath.Join("test_data")
	fileNames, err := ioutil.ReadDir(sourceData)
	require.NoError(t, err)

	rootTmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		sourceData := filepath.Join("test_data", name)
		tmpDir := filepath.Join(rootTmpDir, name)

		t.Logf("running test %s in %s", name, tmpDir)

		err = files.CopyDirOverwrite(sourceData, tmpDir)
		require.NoError(t, err, "failed to copy from %s to %s", sourceData, tmpDir)

		_, o := add.NewCmdAddRepository()
		o.Dir = tmpDir

		o.Args = []string{"https://github.com/jenkins-x/anewthingy.git"}
		err = o.Run()

		testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected.yaml"), filepath.Join(tmpDir, ".jx", "gitops", "source-config.yaml"), "generated source config")
	}

}