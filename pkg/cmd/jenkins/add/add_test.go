package add_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/jenkins/add"
	"github.com/jenkins-x/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJenkinsAdd(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data")

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy from %s to %s", srcDir, tmpDir)

	gitter := cli.NewCLIClient("", nil)
	_, err = gitter.Command(tmpDir, "init")
	require.NoError(t, err, "failed to git init dir %s", tmpDir)

	t.Logf("running test in dir %s\n", tmpDir)

	_, o := add.NewCmdJenkinsAdd()
	o.Dir = tmpDir
	o.Name = "myjenkins"

	err = o.Run()
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	t.Logf("generating to dir %s\n", tmpDir)

	jenkinsDir := filepath.Join(tmpDir, "helmfiles", "myjenkins")
	expectedFile := filepath.Join(jenkinsDir, "helmfile.yaml")
	assert.FileExists(t, expectedFile, "should have generated file")
	t.Logf("generated %s\n", expectedFile)

	sourceConfig, err := sourceconfigs.LoadSourceConfig(tmpDir, false)
	require.NoError(t, err, "failed to load source configs in dir %s", tmpDir)
	require.Len(t, sourceConfig.Spec.JenkinsServers, 1, "should have created 1 jenkins server")
	assert.Equal(t, "myjenkins", sourceConfig.Spec.JenkinsServers[0].Server, "jenkins server name")
}
