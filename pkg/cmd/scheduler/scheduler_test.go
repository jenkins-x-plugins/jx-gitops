package scheduler_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/scheduler"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
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

	testhelpers.AssertConfigMapHasEntry(t, configCM, scheduler.ConfigKey, configFile)
	testhelpers.AssertConfigMapHasEntry(t, pluginsCM, scheduler.PluginsKey, pluginFile)
}
