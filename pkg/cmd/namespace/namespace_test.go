package namespace_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/namespace"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateNamespaceInYamlFiles(t *testing.T) {
	sourceData := filepath.Join("test_data")
	files, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcFile := filepath.Join(sourceData, name, "source.yaml")
			expectedFile := filepath.Join(sourceData, name, "expected.yaml")
			require.FileExists(t, srcFile)
			require.FileExists(t, expectedFile)

			outFile := filepath.Join(tmpDir, name+".yaml")
			err = util.CopyFile(srcFile, outFile)
			require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

			testCases = append(testCases, testCase{
				SourceFile:   srcFile,
				ResultFile:   outFile,
				ExpectedFile: expectedFile,
			})
		}
	}

	err = namespace.UpdateNamespaceInYamlFiles(tmpDir, "something")
	require.NoError(t, err, "failed to update namespace in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := ioutil.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)

		expectData, err := ioutil.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(result, expectedText); d != "" {
			t.Errorf("Generated Pipeline did not match expected: %s", d)
		}
		t.Logf("generated for file %s file %s\n", tc.SourceFile, result)
	}
}
