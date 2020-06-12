package plugins_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/stretchr/testify/assert"
)

func TestHelmPlugin(t *testing.T) {
	t.Parallel()

	plugin := plugins.CreateHelmPlugin("3.1.2")

	assert.Equal(t, plugins.HelmPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.HelmPluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundWindows := false
	for _, b := range plugin.Spec.Binaries {
		if b.Goos == "Linux" && b.Goarch == "amd64" {
			foundLinux = true
			assert.Equal(t, "https://get.helm.sh/helm-v3.1.2-linux-amd64.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		} else if b.Goos == "Windows" && b.Goarch == "amd64" {
			foundWindows = true
			assert.Equal(t, "https://get.helm.sh/helm-v3.1.2-windows-amd64.zip", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}
