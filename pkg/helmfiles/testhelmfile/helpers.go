package testhelmfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// AssertHelmfiles asserts the helmfiles in the given directory versus the output dir
func AssertHelmfiles(t *testing.T, expectedDir string, outDir string, generateTestOutput bool) {
	m := map[string]bool{}
	FindAllHelmfiles(t, m, outDir)
	FindAllHelmfiles(t, m, expectedDir)
	require.NotEmpty(t, m, "failed to find helmfile.yaml files")

	for f := range m {
		outFile := filepath.Join(outDir, f)
		expectedFile := filepath.Join(expectedDir, f)
		if generateTestOutput {
			dir := filepath.Dir(expectedFile)
			err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
			require.NoError(t, err, "failed to make dir %s", dir)

			data, err := ioutil.ReadFile(outFile)
			require.NoError(t, err, "failed to load %s", outFile)

			err = ioutil.WriteFile(expectedFile, data, 0666)
			require.NoError(t, err, "failed to save file %s", expectedFile)
			t.Logf("saved %s\n", expectedFile)
		} else {
			t.Logf("verified %s\n", outFile)
			testhelpers.AssertEqualFileText(t, expectedFile, outFile)
		}
	}
}

// FindAllHelmfiles finds all the relative paths of the helmfiles in the given directory
func FindAllHelmfiles(t *testing.T, m map[string]bool, dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() || info.Name() != "helmfile.yaml" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to get relative path of %s from %s", path, dir)
		}
		m[rel] = true
		return nil
	})
	require.NoError(t, err, "failed to walk dir %s", dir)

}
