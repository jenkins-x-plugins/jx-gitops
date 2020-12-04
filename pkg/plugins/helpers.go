package plugins

import (
	"fmt"
	"os"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetHelmBinary returns the path to the locally installed helm 3 extension
func GetHelmBinary(version string) (string, error) {
	if version == "" {
		version = HelmVersion
	}
	pluginBinDir, err := homedir.PluginBinDir(os.Getenv("JX_GITOPS_HOME"), ".jx-gitops")
	if err != nil {
		return "", errors.Wrapf(err, "failed to find plugin home dir")
	}
	plugin := CreateHelmPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateHelmPlugin creates the helm 3 plugin
func CreateHelmPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		return fmt.Sprintf("https://get.helm.sh/helm-v%s-%s-%s.%s", version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch), p.Extension())
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: HelmPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "helm",
			Binaries:    binaries,
			Description: "helm 3 binary",
			Name:        HelmPluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetHelmfileBinary returns the path to the locally installed helmfile extension
func GetHelmfileBinary(version string) (string, error) {
	if version == "" {
		version = HelmfileVersion
	}
	pluginBinDir, err := homedir.PluginBinDir(os.Getenv("JX_GITOPS_HOME"), ".jx-gitops")
	if err != nil {
		return "", errors.Wrapf(err, "failed to find plugin home dir")
	}
	plugin := CreateHelmfilePlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateHelmfilePlugin creates the helmfile plugin
func CreateHelmfilePlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {

		// workaround until this PR is merged and released it can hopefully be removed:
		//   https://github.com/roboll/helmfile/pull/1612
		if p.Goarch == "arm64" {
			return "https://github.com/jstrachan/helmfile/releases/download/v0.135.0.arm/helmfile_linux_arm64"
		}
		ext := ""
		if p.IsWindows() {
			ext = ".exe"
		}
		return fmt.Sprintf("https://github.com/roboll/helmfile/releases/download/v%s/helmfile_%s_%s%s", version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch), ext)
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: HelmfilePluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "helmfile",
			Binaries:    binaries,
			Description: "helmfile binary",
			Name:        HelmfilePluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetKptBinary returns the path to the locally installed kpt 3 extension
func GetKptBinary(version string) (string, error) {
	if version == "" {
		version = KptVersion
	}
	pluginBinDir, err := homedir.PluginBinDir(os.Getenv("JX_GITOPS_HOME"), ".jx-gitops")
	if err != nil {
		return "", errors.Wrapf(err, "failed to find plugin home dir")
	}
	plugin := CreateKptPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateKptPlugin creates the kpt 3 plugin
func CreateKptPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		return fmt.Sprintf("https://github.com/GoogleContainerTools/kpt/releases/download/v%s/kpt_%s_%s-%s.tar.gz", version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch), version)
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: KptPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "kpt",
			Binaries:    binaries,
			Description: "kpt 3 binary",
			Name:        KptPluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetKubectlBinary returns the path to the locally installed kpt 3 extension
func GetKubectlBinary(version string) (string, error) {
	if version == "" {
		version = KubectlVersion
	}
	pluginBinDir, err := homedir.PluginBinDir(os.Getenv("JX_GITOPS_HOME"), ".jx-gitops")
	if err != nil {
		return "", errors.Wrapf(err, "failed to find plugin home dir")
	}
	plugin := CreateKubectlPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateKubectlPlugin creates the kpt 3 plugin
func CreateKubectlPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		return fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v%s/bin/%s/%s/kubectl", version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch))
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: KubectlPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "kubectl",
			Binaries:    binaries,
			Description: "kubectl binary",
			Name:        KubectlPluginName,
			Version:     version,
		},
	}
	return plugin
}
