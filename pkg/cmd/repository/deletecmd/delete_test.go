package deletecmd_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/deletecmd"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestRepositoryDelete(t *testing.T) {
	testCases := []struct {
		owner, repo, dir string
	}{
		{
			owner: "jenkins-x",
			repo:  "jx-gitops",
			dir:   "simple-owner",
		},
		{
			owner: "",
			repo:  "jx-cli",
			dir:   "simple",
		},
	}
	rootTmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	err = files.CopyDirOverwrite("test_data", rootTmpDir)
	require.NoError(t, err, "failed to copy from test_data to %s", rootTmpDir)

	ns := "jx"
	for _, tc := range testCases {
		name := tc.dir

		tmpDir := filepath.Join(rootTmpDir, name)

		t.Logf("running test %s in %s", name, tmpDir)

		_, o := deletecmd.NewCmdDeleteRepository()
		o.Dir = tmpDir
		o.JXClient = jxfake.NewSimpleClientset()
		o.Namespace = ns
		o.Name = tc.repo
		o.Owner = tc.owner

		err = o.Run()
		require.NoError(t, err, "failed to run")

		expectedPath := filepath.Join("test_data", name, "expected.yaml")
		generatedFile := filepath.Join(tmpDir, ".jx", "gitops", "source-config.yaml")

		if generateTestOutput {
			data, err := os.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = os.WriteFile(expectedPath, data, 0600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		testhelpers.AssertTextFilesEqual(t, expectedPath, generatedFile, "generated source config")
	}
}
