package jobs_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/jenkins/jobs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/maps"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJenkinsJobs(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data", "hasjobs")

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy from %s to %s", srcDir, tmpDir)

	gitter := cli.NewCLIClient("", nil)
	_, err = gitter.Command(tmpDir, "init")
	require.NoError(t, err, "failed to git init dir %s", tmpDir)

	t.Logf("running test in dir %s\n", tmpDir)

	_, o := jobs.NewCmdJenkinsJobs()
	o.Dir = tmpDir

	err = o.Run()
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	t.Logf("generating to dir %s\n", tmpDir)

	jenkinsDir := filepath.Join(tmpDir, "helmfiles", "myjenkins")
	expectedFile := filepath.Join(jenkinsDir, "job-values.yaml")
	assert.FileExists(t, expectedFile, "should have generated file")
	t.Logf("generated %s\n", expectedFile)

	m := map[string]interface{}{}
	err = yamls.LoadFile(expectedFile, &m)
	require.NoError(t, err, "failed to parse YAML file %s", expectedFile)
	path := "controller.JCasC.configScripts.jxsetup"
	script := maps.GetMapValueAsStringViaPath(m, path)
	require.NotEmpty(t, script, "no script populated at path %s", path)
	t.Logf("path %s has script:\n%s\n", path, script)

	assert.FileExists(t, filepath.Join(jenkinsDir, "helmfile.yaml"), "should have created a helmfile.yaml")
}

func TestNoJenkinsJobs(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	_, o := jobs.NewCmdJenkinsJobs()
	o.OutDir = tmpDir
	o.Dir = filepath.Join("test_data", "nojobs")

	err = o.Run()
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)
}
