package create_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/create"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

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
		testhelpers.AssertTextFilesEqual(t, expectedFile, genFile, "generated SourceRepository")

		t.Logf("generated expected file %s\n", genFile)
	}
}
