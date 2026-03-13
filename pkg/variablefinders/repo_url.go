package variablefinders

import (
	"os"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
)

// FindRepositoryURL finds the chart repository URL via environment variables or the dev Environment CRD
func FindRepositoryURL(requirements *jxcore.RequirementsConfig, registryOrg, appName string, oci, pages, explicitlyEmpty bool) string {
	if !explicitlyEmpty {
		answer, exists := os.LookupEnv("JX_CHART_REPOSITORY")
		if !exists && requirements != nil {
			answer = requirements.Cluster.ChartRepository
		}
		if answer != "" {
			return answer
		}
	}
	registry := requirements.Cluster.Registry
	if oci && registryOrg != "" && appName != "" && registry != "" {
		return stringhelpers.UrlJoin(registry, registryOrg, appName)
	}
	if !oci && !pages {
		// assume default chart museum
		return "http://jenkins-x-chartmuseum:8080"
	}
	return ""
}
