package scheduler_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/scheduler"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/lighthouse-client/pkg/config"
	"github.com/jenkins-x/lighthouse-client/pkg/config/keeper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestScheduler(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	require.DirExists(t, sourceDir)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	_, so := scheduler.NewCmdScheduler()

	so.OutDir = tmpDir
	so.Dir = sourceDir

	err = so.Run()
	require.NoError(t, err, "failed to run scheduler command")

	configFile := filepath.Join(tmpDir, scheduler.ConfigMapConfigFileName)
	pluginFile := filepath.Join(tmpDir, scheduler.ConfigMapPluginsFileName)
	require.FileExists(t, configFile)
	require.FileExists(t, pluginFile)

	configCM := &corev1.ConfigMap{}
	pluginsCM := &corev1.ConfigMap{}

	err = yamls.LoadFile(configFile, configCM)
	require.NoError(t, err, "failed to load config file %s", configFile)

	err = yamls.LoadFile(pluginFile, pluginsCM)
	require.NoError(t, err, "failed to load config file %s", pluginFile)

	yamlText := testhelpers.AssertConfigMapHasEntry(t, configCM, scheduler.ConfigKey, configFile)
	ym := AssertYamlMap(t, yamlText, configFile)
	for _, k := range []string{"branch-protection", "postsubmits", "presubmits", "tide"} {
		assert.Contains(t, ym, k, configFile)
	}

	yamlText = testhelpers.AssertConfigMapHasEntry(t, pluginsCM, scheduler.PluginsKey, pluginFile)
	ym = AssertYamlMap(t, yamlText, pluginFile)
	for _, k := range []string{"approve", "plugins", "triggers"} {
		assert.Contains(t, ym, k, pluginFile)
	}

	// lets load the LH config
	configYaml := configCM.Data["config.yaml"]
	require.NotEmpty(t, configYaml, "no config.yaml in generated ConfigMap")

	lhCfg, err := config.LoadYAMLConfig([]byte(configYaml))
	require.NoError(t, err, "failed to load config file %s into lighthouse config", configFile)

	repoName := "myorg/default"
	assert.Len(t, lhCfg.Presubmits[repoName], 1, "presubmits for %s", repoName)
	assert.Len(t, lhCfg.Postsubmits[repoName], 1, "postsubmits for %s", repoName)

	repoName = "myorg/env-mycluster-dev"
	assert.Len(t, lhCfg.Presubmits[repoName], 0, "presubmits for %s", repoName)
	assert.Len(t, lhCfg.Postsubmits[repoName], 0, "postsubmits for %s", repoName)

	inRepoFullName := "myorg/in-repo"
	otherInRepoFullName := "myorg/another-in-repo"
	for _, repoName := range []string{inRepoFullName} {
		assert.Len(t, lhCfg.Presubmits[repoName], 0, "presubmits for %s", repoName)
		assert.Len(t, lhCfg.Postsubmits[repoName], 0, "postsubmits for %s", repoName)
	}

	assert.NotNil(t, lhCfg.InRepoConfig.Enabled, "should have inRepoConfig enabled")
	assert.NotNil(t, lhCfg.InRepoConfig.Enabled[inRepoFullName], "should have inRepoConfig.ToBool['myorg/in-repo']")
	assert.NotNil(t, lhCfg.InRepoConfig.Enabled[otherInRepoFullName], "should have inRepoConfig.ToBool['myorg/another-in-repo']")
	assert.NotNil(t, lhCfg.InRepoConfig.Enabled["myorg/env-mycluster-dev"], "should have inRepoConfig.ToBool['myorg/env-mycluster-dev']")
	assert.NotNil(t, lhCfg.InRepoConfig.Enabled["jxbdd/myrepo"], "should have inRepoConfig.ToBool['jxbdd/myrepo']")
	assert.NotNil(t, lhCfg.InRepoConfig.Enabled["JXBDD/myrepo"], "should have inRepoConfig.ToBool['JXBDD/myrepo']")

	approveQuery := keeper.Query{}
	foundApproveQuery := false
	for _, q := range lhCfg.Keeper.Queries {
		if stringhelpers.StringArrayIndex(q.Repos, inRepoFullName) >= 0 && stringhelpers.StringArrayIndex(q.Labels, "approved") >= 0 {
			approveQuery = q
			foundApproveQuery = true
			break
		}
	}
	require.True(t, foundApproveQuery, "no approve query found for repo %s ", inRepoFullName)

	requiredMissingLabel := "do-not-merge"

	assert.True(t, stringhelpers.StringArrayIndex(approveQuery.MissingLabels, requiredMissingLabel) >= 0, "should have a missing label %s", requiredMissingLabel)

	assert.Equal(t, "http://deck-jx..jx.1.2.3.4.nip.io", lhCfg.Keeper.TargetURL, "config.Keeper.TargetURL")
}

func AssertYamlMap(t *testing.T, text, message string) map[string]interface{} {
	require.NotEmpty(t, text, "no YAML text for %s", message)

	m := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(text), &m)
	require.NoError(t, err, "failed to parse YAML %s for %s", text, message)
	return m
}
