package variablefinders

import (
	"os"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
)

// FindRepositoryURL finds the chart repository URL via environment variables or the dev Environment CRD
func FindRepositoryURL(jxClient jxc.Interface, ns string, requirements *jxcore.RequirementsConfig, registryOrg, appName string) (string, error) {
	answer := ""
	if requirements != nil {
		answer = requirements.Cluster.ChartRepository
	}
	if answer == "" {
		answer = os.Getenv("JX_CHART_REPOSITORY")
	}
	if answer == "" {
		registry := requirements.Cluster.Registry
		if requirements.Cluster.ChartOCI && registryOrg != "" && appName != "" && registry != "" {
			return stringhelpers.UrlJoin(registry, registryOrg, appName), nil
		}
		// assume default chart museum
		answer = "http://jenkins-x-chartmuseum:8080"
	}
	return answer, nil
}
