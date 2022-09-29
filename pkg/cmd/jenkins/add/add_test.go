package add_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/resolve"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/jenkins/add"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJenkinsAdd(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := "testdata"

	runner := &fakerunner.FakeRunner{}

	err := files.CopyDirOverwrite(srcDir, tmpDir)
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

	assert.FileExists(t, filepath.Join(jenkinsDir, "values.yaml"), "should have generated file")

	sourceConfig, err := sourceconfigs.LoadSourceConfig(tmpDir, false)
	require.NoError(t, err, "failed to load source configs in dir %s", tmpDir)
	require.Len(t, sourceConfig.Spec.JenkinsServers, 1, "should have created 1 jenkins server")
	assert.Equal(t, "myjenkins", sourceConfig.Spec.JenkinsServers[0].Server, "jenkins server name")

	assertValidHelmfile(t, expectedFile)

	// lets test that re-running the command doesn't result in duplicates
	_, o = add.NewCmdJenkinsAdd()
	o.Dir = tmpDir
	o.Name = "myjenkins"

	err = o.Run()
	require.NoError(t, err, "failed to run the command in dir %s", tmpDir)

	assertValidHelmfile(t, expectedFile)

	// now lets run helmfile resolve...
	_, ro := resolve.NewCmdHelmfileResolve()
	ro.Dir = tmpDir
	ro.QuietCommandRunner = runner.Run
	ro.CommandRunner = runner.Run
	ro.TestOutOfCluster = true

	err = ro.Run()
	require.NoError(t, err, "failed to run the helmfile resolve in dir %s", tmpDir)

	assertValidHelmfile(t, expectedFile)
}

func assertValidHelmfile(t *testing.T, expectedFile string) {
	helmState := &state.HelmState{}
	err := yaml2s.LoadFile(expectedFile, helmState)
	require.NoError(t, err, "failed to load %s", expectedFile)

	AssertHemlfileChartCount(t, 1, helmState, "jxgh/jenkins-resources", "file %s", expectedFile)
	AssertHemlfileChartCount(t, 1, helmState, "jenkinsci/jenkins", "file %s", expectedFile)

	AssertHemlfileRepository(t, helmState, "jenkinsci", "https://charts.jenkins.io", "file %s", expectedFile)
	AssertHemlfileRepository(t, helmState, "jxgh", helmhelpers.JX3HelmRepository, "file %s", expectedFile)
}

func AssertHemlfileChartCount(t *testing.T, expectedCount int, helmState *state.HelmState, chartName string, messageAndArgs ...interface{}) {
	count := 0
	for k := range helmState.Releases {
		rel := helmState.Releases[k]
		if rel.Chart == chartName {
			count++
		}
	}
	assert.Equal(t, expectedCount, count, messageAndArgs...)
}

func AssertHemlfileRepository(t *testing.T, helmState *state.HelmState, name, url, message string, args ...interface{}) {
	text := fmt.Sprintf(message, args...)
	found := false
	for k := range helmState.Repositories {
		r := helmState.Repositories[k]
		if r.Name == name {
			found = true
			assert.Equal(t, url, r.URL, "url for repository name %s for %s", name, text)
		}
	}
	assert.True(t, found, "should have found repository with name %s url %s for %s", name, url, text)
}
