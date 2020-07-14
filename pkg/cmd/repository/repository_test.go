package repository_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/repository"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRepositorySourceDir(t *testing.T) {
	sourceData := filepath.Join("test_data", "sourcedir")
	fileNames, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	t.Logf("generating fileNames to %s\n", tmpDir)

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()
			srcFile := filepath.Join(sourceData, name, "source.yaml")
			expectedFile := filepath.Join(sourceData, name, "expected.yaml")
			require.FileExists(t, srcFile)
			require.FileExists(t, expectedFile)

			outFile := filepath.Join(tmpDir, name+".yaml")
			err = files.CopyFile(srcFile, outFile)
			require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

			testCases = append(testCases, testCase{
				SourceFile:   srcFile,
				ResultFile:   outFile,
				ExpectedFile: expectedFile,
			})
		}
	}

	_, o := repository.NewCmdUpdateRepository()
	o.Dir = tmpDir
	o.SourceDir = tmpDir

	gitURL := "https://github.com/someorg/somerepo.git"

	err = o.Run([]string{gitURL})
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := ioutil.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)

		expectData, err := ioutil.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(result, expectedText); d != "" {
			t.Errorf("Generated Pipeline for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestUpdateRepositoryInRequirements(t *testing.T) {
	srcFile := filepath.Join("test_data", "requirements", "jx-requirements.yml")

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	outFile := filepath.Join(tmpDir, "jx-requirements.yml")
	err = files.CopyFile(srcFile, outFile)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)
	require.FileExists(t, outFile)

	t.Logf("modifying requirements file  to %s\n", outFile)

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	_, o := repository.NewCmdUpdateRepository()
	o.Dir = tmpDir
	o.SourceDir = tmpDir

	gitURL := "https://github.com/someorg/somerepo.git"

	err = o.Run([]string{gitURL})
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	requirements, _, err := config.LoadRequirementsConfig(tmpDir, true)
	require.NoError(t, err, "failed to load requirements file %s", outFile)
	require.NotNil(t, requirements, "no requirements file %s", outFile)

	found := false
	for _, env := range requirements.Environments {
		if env.Key == "dev" {
			found = true
			assert.Equal(t, "someorg", env.Owner, "owner for dev env in file %s", outFile)
			assert.Equal(t, "somerepo", env.Repository, "repo for dev env in file %s", outFile)
		}
	}
	assert.True(t, found, "not found a 'dev' environment in the requirement file %s", outFile)

}
