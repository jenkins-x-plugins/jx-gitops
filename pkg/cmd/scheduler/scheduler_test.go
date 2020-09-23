package scheduler_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/scheduler"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/jenkins-x/lighthouse/pkg/config"
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

	for _, repoName := range []string{"myorg/default", "myorg/env-mycluster-dev"} {
		assert.Len(t, lhCfg.Presubmits[repoName], 1, "presubmits for %s", repoName)
		assert.Len(t, lhCfg.Postsubmits[repoName], 1, "postsubmits for %s", repoName)
	}

	for _, repoName := range []string{"myorg/in-repo"} {
		assert.Len(t, lhCfg.Presubmits[repoName], 0, "presubmits for %s", repoName)
		assert.Len(t, lhCfg.Postsubmits[repoName], 0, "postsubmits for %s", repoName)
	}

	assert.NotEmpty(t, lhCfg.InRepoConfig.Enabled, "should have inRepoConfig enabled")

	assert.Equal(t, "http://deck-jx..jx.1.2.3.4.nip.io", lhCfg.Keeper.TargetURL, "config.Keeper.TargetURL")
}

func AssertYamlMap(t *testing.T, text string, message string) map[string]interface{} {
	require.NotEmpty(t, text, "no YAML text for %s", message)

	m := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(text), &m)
	require.NoError(t, err, "failed to parse YAML %s for %s", text, message)
	return m
}
