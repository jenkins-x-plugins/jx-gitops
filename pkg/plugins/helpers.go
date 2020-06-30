package plugins

import (
	"fmt"
	"os"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/pkg/homedir"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// HelmPluginName the default name of the helm plugin
	HelmPluginName = "helm"
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
