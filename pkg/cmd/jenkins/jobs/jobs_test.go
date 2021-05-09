package jobs_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/jenkins/add"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/jenkins/jobs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/maps"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jenkinsName = "myjenkins"

func TestJenkinsJobs(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data", "hasjobs")

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy from %s to %s", srcDir, tmpDir)

	gitter := cli.NewCLIClient("", nil)
	_, err = gitter.Command(tmpDir, "init")
	require.NoError(t, err, "failed to git init dir %s", tmpDir)

	AssertGenerateJobs(t, tmpDir, jenkinsName)
}

func TestJenkinsJobsForExistingJenkins(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data", "hasjobs")

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy from %s to %s", srcDir, tmpDir)

	gitter := cli.NewCLIClient("", nil)
	_, err = gitter.Command(tmpDir, "init")
	require.NoError(t, err, "failed to git init dir %s", tmpDir)

	// lets add a jenkins server...
	_, ao := add.NewCmdJenkinsAdd()
	ao.Name = jenkinsName
	ao.Dir = tmpDir
	err = ao.Run()
	require.NoError(t, err, "failed to run the add jenkins command in dir %s", tmpDir)

	AssertGenerateJobs(t, tmpDir, jenkinsName)
}

func AssertGenerateJobs(t *testing.T, tmpDir string, jenkinsName string) {
	t.Logf("running test in dir %s\n", tmpDir)

	_, o := jobs.NewCmdJenkinsJobs()
	o.Dir = tmpDir

	err := o.Run()
	require.NoError(t, err, "failed to run the generate jobs command in dir %s", tmpDir)

	t.Logf("generating to dir %s\n", tmpDir)

	jenkinsDir := filepath.Join(tmpDir, "helmfiles", jenkinsName)
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

	jenkinsHelmfile := filepath.Join(jenkinsDir, "helmfile.yaml")
	assert.FileExists(t, jenkinsHelmfile, "should have created a helmfile.yaml")

	state := &state.HelmState{}
	err = yaml2s.LoadFile(jenkinsHelmfile, state)
	require.NoError(t, err, "failed to load jenkins helmfile %s", jenkinsHelmfile)
	require.NotEmpty(t, state.Releases, "no releases in %s", jenkinsHelmfile)

	release := state.Releases[0]
	assert.Equal(t, jenkinsName, state.OverrideNamespace, "namespace for %s", jenkinsHelmfile)
	assert.Equal(t, "", release.Namespace, "release.Namespace for %s", jenkinsHelmfile)
	assert.Equal(t, "jenkins", release.Name, "release.Name for %s", jenkinsHelmfile)
	assert.Equal(t, "jenkinsci/jenkins", release.Chart, "release.Chart for %s", jenkinsHelmfile)

	assert.Contains(t, release.Values, "job-values.yaml", "for jenkins release in %s", jenkinsHelmfile)
	assert.Contains(t, release.Values, "values.yaml", "for jenkins release in %s", jenkinsHelmfile)
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
