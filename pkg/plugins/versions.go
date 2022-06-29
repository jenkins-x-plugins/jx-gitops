package plugins

import jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"

const (
	// HelmPluginName the default name of the helm plugin
	HelmPluginName = "helm"

	// HelmfilePluginName the default name of the helmfile plugin
	HelmfilePluginName = "helmfile"

	// KptPluginName the default name of the kpt plugin
	KptPluginName = "kpt"

	// KubectlPluginName the default name of the kubectl plugin
	KubectlPluginName = "kubectl"

	// KappPluginName the default name of the kapp plugin
	KappPluginName = "kapp"

	// KustomizePluginName the default name of the kustomize plugin
	KustomizePluginName = "kustomize"

	// HelmVersion the default version of helm to use
	HelmVersion = "3.7.2"

	// HelmfileVersion the default version of helmfile to use
	HelmfileVersion = "0.143.0"

	// KptVersion the default version of kpt to use
	KptVersion = "1.0.0-beta.17"

	// KubectlVersion the default version of kpt to use
	KubectlVersion = "1.21.0"

	// KappVersion the default version of kapp to use
	KappVersion = "0.35.1-cmfork"

	// KustomizeVersion the default version of kustomize to use
	KustomizeVersion = "4.4.1"
)

type HelmPlugin struct {
	URL  string
	Name string
}

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		CreateHelmPlugin(HelmVersion),
		CreateHelmfilePlugin(HelmfileVersion),
		CreateKptPlugin(KptVersion),
		CreateKubectlPlugin(KubectlVersion),
		CreateKappPlugin(KappVersion),
		CreateKustomizePlugin(KustomizeVersion),
	}

	// HelmPlugins to install and upgrade
	HelmPlugins = []HelmPlugin{
		{"https://github.com/mumoshu/helm-x", "x"},
		{"https://github.com/hypnoglow/helm-s3", "s3"},
	}
)
