package label_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/tagging"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateLabelsInYamlFiles(t *testing.T) {
	sourceData := "testdata"
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	argTests := [][]string{
		{"chart-", "beer=stella", "wine=merlot"},
		{"wine=merlot", "beer=stella", "chart-"},
	}

	for _, args := range argTests {
		tmpDir := t.TempDir()
		tmpDirNotOverride := t.TempDir()

		type testCase struct {
			SourceFile              string
			ResultFile              string
			ResultNotOverrideFile   string
			ExpectedFile            string
			ExpectedNotOverrideFile string
		}

		var testCases []testCase
		for _, f := range fileNames {
			if !f.IsDir() {
				continue
			}

			name := f.Name()
			srcFile := filepath.Join(sourceData, name, "source.yaml")
			expectedFile := filepath.Join(sourceData, name, "expected.yaml")
			expectedNotOverrideFile := filepath.Join(sourceData, name, "expectednotoverride.yaml")

			require.FileExists(t, srcFile)
			require.FileExists(t, expectedFile)
			require.FileExists(t, expectedNotOverrideFile)

			outFile := filepath.Join(tmpDir, name+".yaml")
			err = files.CopyFile(srcFile, outFile)
			require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

			outFileNotOverride := filepath.Join(tmpDirNotOverride, name+".yaml")
			err = files.CopyFile(srcFile, outFileNotOverride)
			require.NoError(t, err, "failed to copy %s to %s", srcFile, outFileNotOverride)

			testCases = append(testCases, testCase{
				SourceFile:              srcFile,
				ResultFile:              outFile,
				ResultNotOverrideFile:   outFileNotOverride,
				ExpectedFile:            expectedFile,
				ExpectedNotOverrideFile: expectedNotOverrideFile,
			})

		}
		err = tagging.UpdateTagInYamlFiles(tmpDir, "labels", args, kyamls.Filter{}, false, true)
		require.NoError(t, err, "failed to update namespace in dir %s for args %#v", tmpDir, args)
		err = tagging.UpdateTagInYamlFiles(tmpDirNotOverride, "labels", args, kyamls.Filter{}, false, false)
		require.NoError(t, err, "failed to update namespace in dir %s for args %#v", tmpDir, args)

		for _, tc := range testCases {
			resultData, err := os.ReadFile(tc.ResultFile)
			require.NoError(t, err, "failed to load results %s for args %#v", tc.ResultFile, args)

			expectData, err := os.ReadFile(tc.ExpectedFile)
			require.NoError(t, err, "failed to load results %s for args %#v", tc.ExpectedFile, args)

			result := strings.TrimSpace(string(resultData))
			expectedText := strings.TrimSpace(string(expectData))
			if d := cmp.Diff(result, expectedText); d != "" {
				t.Errorf("Generated Pipeline for file %s did not match expected: %s for args %#v", tc.SourceFile, d, args)
			}

			t.Logf("generated for file %s with args %#v file\n%s\n", tc.SourceFile, args, result)

			resultData, err = os.ReadFile(tc.ResultNotOverrideFile)
			require.NoError(t, err, "failed to load results %s for args %#v", tc.ResultFile, args)

			expectData, err = os.ReadFile(tc.ExpectedNotOverrideFile)
			require.NoError(t, err, "failed to load results %s for args %#v", tc.ExpectedFile, args)

			result = strings.TrimSpace(string(resultData))
			expectedText = strings.TrimSpace(string(expectData))
			if d := cmp.Diff(result, expectedText); d != "" {
				t.Errorf("Generated Pipeline for file %s did not match expected: %s for args --override=false %#v", tc.SourceFile, d, args)
			}

			t.Logf("generated for file %s with args --override=false %#v file\n%s\n", tc.SourceFile, args, result)
		}
	}
}
