package jobs_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/jenkins/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJenkinsJobs(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	_, o := jobs.NewCmdJenkinsJobs()
	o.OutDir = tmpDir
	o.Dir = filepath.Join("test_data", "hasjobs")

	err = o.Run()
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	t.Logf("generating to dir %s\n", tmpDir)

	expectedFile := filepath.Join(tmpDir, "myjenkins", "values.yaml")
	assert.FileExists(t, expectedFile, "should have generated file")
	t.Logf("generated %s\n", expectedFile)
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
