package ingress_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/ingress"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/require"
)

// generateTestOutput enable to regenerate the expected output
var generateTestOutput = false

func TestUpdateIngressNoTLS(t *testing.T) {
	AssertUpdateIngress(t, filepath.Join("testdata", "notls"))
}

func TestUpdateIngressTLS(t *testing.T) {
	AssertUpdateIngress(t, filepath.Join("testdata", "tls"))
}

func AssertUpdateIngress(t *testing.T, rootDir string) {
	require.DirExists(t, rootDir)
	sourceData := filepath.Join(rootDir, "source")
	require.DirExists(t, sourceData)

	expectedData := filepath.Join(rootDir, "expected")
	require.DirExists(t, expectedData)

	tmpDir := t.TempDir()

	err := files.CopyDir(sourceData, tmpDir, true)
	require.NoError(t, err, "failed to copy from %s to %s", sourceData, tmpDir)

	_, uo := ingress.NewCmdUpdateIngress()
	uo.Dir = tmpDir
	err = uo.Run()
	require.NoError(t, err, "failed to run update ingress in dir %s", tmpDir)

	// now lets compare files with expected
	sourceConfigDir := filepath.Join(tmpDir, "config-root")
	err = filepath.Walk(sourceConfigDir, func(path string, info os.FileInfo, err error) error { //nolint:staticcheck
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		rel, err := filepath.Rel(sourceConfigDir, path) //nolint:staticcheck
		require.NoError(t, err)

		expectedFile := filepath.Join(expectedData, rel)

		require.FileExists(t, path)
		require.FileExists(t, expectedFile)

		resultData, err := os.ReadFile(path)
		require.NoError(t, err, "failed to load results %s", path)

		expectData, err := os.ReadFile(expectedFile)
		require.NoError(t, err, "failed to load results %s", expectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))

		if generateTestOutput {
			err = os.WriteFile(expectedFile, []byte(result), 0o600)
			require.NoError(t, err, "failed to save file %s", expectedFile)
			return nil
		}
		if d := cmp.Diff(result, expectedText); d != "" {
			t.Errorf("modified file %s match expected: %s", path, d)
		}
		t.Logf("found file %s file %s\n", path, result)
		return nil
	})
	require.NoError(t, err, "failed to process")
}
