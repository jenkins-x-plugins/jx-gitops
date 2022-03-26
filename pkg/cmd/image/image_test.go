package image_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/image"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestUpdateImages(t *testing.T) {
	_, o := image.NewCmdUpdateImage()

	inputDir := filepath.Join("test_data", "input")
	expectedDir := filepath.Join("test_data", "expected")
	require.DirExists(t, inputDir)
	require.DirExists(t, expectedDir)

	tmpDir := t.TempDir()

	err := files.CopyDirOverwrite(inputDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", inputDir, tmpDir)

	o.SourceDir = filepath.Join(tmpDir, "src")
	o.VersionStreamer.Dir = tmpDir

	t.Logf("modifying files at %s\n", o.SourceDir)

	err = o.Run()
	require.NoError(t, err, "failed to convert images")

	// lets assert the files match the expected
	err = filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error { //nolint:staticcheck
		if info == nil || info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(expectedDir, path) //nolint:staticcheck
		if err != nil {
			return errors.Wrapf(err, "failed to find relative path of %s", path)
		}

		t.Logf("comparing to expected file %s\n", path)
		testhelpers.AssertTextFilesEqual(t, path, filepath.Join(tmpDir, relPath), "output")
		return nil
	})
	require.NoError(t, err, "failed to walk expected files")
}
