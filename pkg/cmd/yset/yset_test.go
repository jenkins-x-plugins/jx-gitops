package yset_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/yset"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

// generateTestOutput enable to regenerate the expected output
var generateTestOutput = false

func TestYSet(t *testing.T) {
	testCases := []struct {
		dir   string
		path  string
		value string
	}{
		{
			dir:   "image_tag",
			path:  "image.tag",
			value: "1.2.3",
		},
		{
			dir:   "top_level",
			path:  "replicaCount",
			value: "10",
		},
		{
			dir:   "missing_top_level",
			path:  "myTopLevel",
			value: "beer wine",
		},
	}

	tmpDir := t.TempDir()

	for _, tc := range testCases {
		name := tc.dir

		srcFile := filepath.Join("test_data", name, "source.yaml")
		expectedFile := filepath.Join("test_data", name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err := files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		_, o := yset.NewCmdYSet()

		o.Files = []string{outFile}
		o.Path = tc.path
		o.Value = tc.value

		err = o.Run()
		require.NoError(t, err, "failed to run for test %s", name)

		if generateTestOutput {
			data, err := ioutil.ReadFile(outFile)
			require.NoError(t, err, "failed to load %s", outFile)

			err = ioutil.WriteFile(expectedFile, data, 0600)
			require.NoError(t, err, "failed to save file %s", expectedFile)
			continue
		}
		_ = testhelpers.AssertEqualFileText(t, expectedFile, outFile)
	}
}
