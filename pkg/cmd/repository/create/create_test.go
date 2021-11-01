package create_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/create"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/v4/pkg/util"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

// generateTestOutput enable to regenerate the expected output
var generateTestOutput = false

func TestCreateRepositorySourceDir(t *testing.T) {
	sourceData := filepath.Join("test_data", "input")

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	t.Logf("generating SourceRepository files in %s", tmpDir)

	err = files.CopyDirOverwrite(sourceData, tmpDir)
	require.NoError(t, err, "failed to copy from %s to %s", sourceData, tmpDir)

	_, o := create.NewCmdCreateRepository()
	o.Dir = tmpDir

	err = o.Run()
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	expectedDir := filepath.Join("test_data", "expected", "config-root", "namespaces", "jx", "source-repositories")
	genDir := filepath.Join(tmpDir, "config-root", "namespaces", "jx", "source-repositories")

	for _, name := range []string{"jenkins-x-jx-cli.yaml", "jenkins-x-jx-gitops.yaml", "mygitlaborg-somegitlab.yaml"} {
		expectedFile := filepath.Join(expectedDir, name)
		genFile := filepath.Join(genDir, name)

		if generateTestOutput {
			generatedFile := genFile
			expectedPath := expectedFile
			data, err := ioutil.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = ioutil.WriteFile(expectedPath, data, 0600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		testhelpers.AssertTextFilesEqual(t, expectedFile, genFile, "generated SourceRepository")

		t.Logf("generated expected file %s\n", genFile)

		target := &v1.SourceRepository{}
		data, err := ioutil.ReadFile(genFile)
		require.NoError(t, err, "failed to read file %s", genFile)

		results, err := util.ValidateYaml(target, data)
		require.NoError(t, err, "failed to validate file %s", genFile)
		require.Empty(t, results, "should have validated file %s", genFile)
	}
}
