package extsecret_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/extsecret"
	"github.com/jenkins-x/jx-gitops/pkg/secretmapping"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToExtSecrets(t *testing.T) {
	sourceData := filepath.Join("test_data", "simple")
	fileNames, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()
			if name == ".jx" {
				continue
			}
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

	_, eo := extsecret.NewCmdExtSecrets()
	eo.Dir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

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

func TestInvalidSecretMapping(t *testing.T) {
	t.Skip("see https://github.com/jenkins-x/jx-gitops/issues/15")
	_, eo := extsecret.NewCmdExtSecrets()
	var err error
	sourceData := filepath.Join("test_data", "invalid_secret_mappings")
	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.Error(t, err, "failed did not receive error validating missing backend type")
}

func TestMultipleBackendTypes(t *testing.T) {
	sourceData := filepath.Join("test_data", "backend_types")
	fileNames, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()
			if name == ".jx" {
				continue
			}
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

	_, eo := extsecret.NewCmdExtSecrets()
	eo.Dir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	assert.Equal(t, v1alpha1.BackendTypeVault, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeGSM, eo.SecretMapping.Spec.Secrets[1].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

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
