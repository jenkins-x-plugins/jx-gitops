//nolint:dupl
package plugins_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHelmfilePlugin(t *testing.T) {
	t.Parallel()

	plugin := plugins.CreateHelmfilePlugin(plugins.HelmfileVersion)

	assert.Equal(t, plugins.HelmfilePluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.HelmfilePluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundMac := false
	foundWindows := false
	foundArm := false
	for _, b := range plugin.Spec.Binaries {
		switch b.Goarch {
		case "arm64":
			if b.Goos == "Linux" {
				foundArm = true
				assert.Equal(t, "https://github.com/helmfile/helmfile/releases/download/v"+plugins.HelmfileVersion+"/helmfile_"+plugins.HelmfileVersion+"_linux_arm64.tar.gz", b.URL, "URL for linux arm binary")
				t.Logf("found linux binary URL %s", b.URL)
			}

		case "amd64":
			switch b.Goos {
			case "Darwin":
				foundMac = true
				assert.Equal(t, "https://github.com/helmfile/helmfile/releases/download/v"+plugins.HelmfileVersion+"/helmfile_"+plugins.HelmfileVersion+"_darwin_amd64.tar.gz", b.URL, "URL for linux binary")
				t.Logf("found mac binary URL %s", b.URL)
			case "Linux":
				foundLinux = true
				assert.Equal(t, "https://github.com/helmfile/helmfile/releases/download/v"+plugins.HelmfileVersion+"/helmfile_"+plugins.HelmfileVersion+"_linux_amd64.tar.gz", b.URL, "URL for linux binary")
				t.Logf("found linux binary URL %s", b.URL)
			case "Windows":
				foundWindows = true
				assert.Equal(t, "https://github.com/helmfile/helmfile/releases/download/v"+plugins.HelmfileVersion+"/helmfile_"+plugins.HelmfileVersion+"_windows_amd64.tar.gz", b.URL, "URL for windows binary")
				t.Logf("found windows binary URL %s", b.URL)
			}
		}
	}
	assert.True(t, foundArm, "did not find an arm linux binary in the plugin %#v", plugin)
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
			assert.Equal(t, "https://github.com/GoogleContainerTools/kpt/releases/download/v"+v+"/kpt_darwin_amd64-"+v+".tar.gz", b.URL, "URL for linux binary")
			t.Logf("found mac binary URL %s", b.URL)
		case "Linux":
			foundLinux = true
			assert.Equal(t, "https://github.com/GoogleContainerTools/kpt/releases/download/v"+v+"/kpt_linux_amd64-"+v+".tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		case "Windows":
			foundWindows = true
			assert.Equal(t, "https://github.com/GoogleContainerTools/kpt/releases/download/v"+v+"/kpt_windows_amd64-"+v+".tar.gz", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundMac, "did not find a mac binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}

func TestKappPlugin(t *testing.T) {
	t.Parallel()

	v := plugins.KappVersion
	plugin := plugins.CreateKappPlugin(v)

	assert.Equal(t, plugins.KappPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.KappPluginName, plugin.Spec.Name, "plugin.Spec.Name")

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

			assert.Equal(t, "https://github.com/chrismellard/carvel-kapp/releases/download/v"+v+"/carvel-kapp_"+v+"_Darwin_amd64.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found mac binary URL %s", b.URL)
		case "Linux":
			foundLinux = true
			assert.Equal(t, "https://github.com/chrismellard/carvel-kapp/releases/download/v"+v+"/carvel-kapp_"+v+"_Linux_amd64.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		case "Windows":
			foundWindows = true
			assert.Equal(t, "https://github.com/chrismellard/carvel-kapp/releases/download/v"+v+"/carvel-kapp_"+v+"_Windows_amd64.tar.gz", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundMac, "did not find a mac binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}

func TestKubectlPlugin(t *testing.T) {
	t.Parallel()

	v := plugins.KubectlVersion
	plugin := plugins.CreateKubectlPlugin(v)

	assert.Equal(t, plugins.KubectlPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.KubectlPluginName, plugin.Spec.Name, "plugin.Spec.Name")

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
			assert.Equal(t, "https://storage.googleapis.com/kubernetes-release/release/v"+v+"/bin/darwin/amd64/kubectl", b.URL, "URL for linux binary")
			t.Logf("found mac binary URL %s", b.URL)
		case "Linux":
			foundLinux = true
			assert.Equal(t, "https://storage.googleapis.com/kubernetes-release/release/v"+v+"/bin/linux/amd64/kubectl", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		case "Windows":
			foundWindows = true
			assert.Equal(t, "https://storage.googleapis.com/kubernetes-release/release/v"+v+"/bin/windows/amd64/kubectl", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundMac, "did not find a mac binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}

func TestPluginDir(t *testing.T) {
	testCases := []struct {
		env      map[string]string
		expected string
	}{
		{
			env: map[string]string{
				"JX_GITOPS_HOME": "/tmp/root/.my-jx-gitops",
			},
			expected: "/tmp/root/.my-jx-gitops/plugins/bin",
		},
		{
			env: map[string]string{
				"JX_HOME": "/tmp/home/.jx",
			},
			expected: "/tmp/home/.jx/plugins/bin",
		},
		{
			env: map[string]string{
				"JX3_HOME": "/tmp/home/.jx3",
			},
			expected: "/tmp/home/.jx3/plugins/bin",
		},
		{
			env:      map[string]string{},
			expected: filepath.Join(homedir.HomeDir(), ".jx", "plugins", "bin"),
		},
	}

	for _, tc := range testCases {
		fn := func(k string) string {
			return tc.env[k]
		}
		dir, err := plugins.PluginBinDirFunc(fn)
		require.NoError(t, err, "failed to get plugin dir")
		t.Logf("got plugin dir %s\n", dir)
		assert.Equal(t, tc.expected, dir, "for env %v", tc.env)
	}
}
