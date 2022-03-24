package escape_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helm/escape"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestEscapeYAML(t *testing.T) {
	srcDir := filepath.Join("test_data", "src")
	require.DirExists(t, srcDir)

	tmpDir := t.TempDir()

	err := files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

	_, o := escape.NewCmdEscape()
	o.Dir = tmpDir

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcDir, tmpDir)

	t.Logf("escape files in dir %s\n", tmpDir)

	srcFile := filepath.Join(tmpDir, "config-observability-cm.yaml")
	expectedFile := filepath.Join("test_data", "expected", "config-observability-cm.yaml")
	_ = testhelpers.AssertEqualFileText(t, expectedFile, srcFile)
}
