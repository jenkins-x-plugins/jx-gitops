package plugins_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/stretchr/testify/assert"
)

func TestHelmPlugin(t *testing.T) {
	t.Parallel()

	plugin := plugins.CreateHelmPlugin(plugins.HelmVersion)

	assert.Equal(t, plugins.HelmPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.HelmPluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundMac := false
	foundWindows := false
	for _, b := range plugin.Spec.Binaries {
		if b.Goarch != "amd64" {
			continue
		}
		switch b.Goos {
		case "Darwin":
			foundMac = true
			assert.Equal(t, "https://get.helm.sh/helm-v"+plugins.HelmVersion+"-darwin-amd64.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found mac binary URL %s", b.URL)
		case "Linux":
			foundLinux = true
			assert.Equal(t, "https://get.helm.sh/helm-v"+plugins.HelmVersion+"-linux-amd64.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		case "Windows":
			foundWindows = true
			assert.Equal(t, "https://get.helm.sh/helm-v"+plugins.HelmVersion+"-windows-amd64.zip", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundMac, "did not find a mac binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}

func TestKptPlugin(t *testing.T) {
	t.Parallel()

	v := plugins.KptVersion
	plugin := plugins.CreateKptPlugin(v)

	assert.Equal(t, plugins.KptPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.KptPluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundMac := false
	foundWindows := false
	for _, b := range plugin.Spec.Binaries {
		if b.Goarch != "amd64" {
			continue
		}
		switch b.Goos {
		case "Darwin":
			foundMac = true
			assert.Equal(t, "https://github.com/GoogleContainerTools/kpt/releases/download/v"+v+"/kpt_darwin_amd64_"+v+".tar.gz", b.URL, "URL for linux binary")
			t.Logf("found mac binary URL %s", b.URL)
		case "Linux":
			foundLinux = true
			assert.Equal(t, "https://github.com/GoogleContainerTools/kpt/releases/download/v"+v+"/kpt_linux_amd64_"+v+".tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		case "Windows":
			foundWindows = true
			assert.Equal(t, "https://github.com/GoogleContainerTools/kpt/releases/download/v"+v+"/kpt_windows_amd64_"+v+".tar.gz", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundMac, "did not find a mac binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}
