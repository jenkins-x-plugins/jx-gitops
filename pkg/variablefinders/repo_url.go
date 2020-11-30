package variablefinders

import (
	"os"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
)

// FindRepositoryURL finds the chart repository URL via environment variables or the dev Environment CRD
func FindRepositoryURL(jxClient jxc.Interface, ns string, requirements *jxcore.RequirementsConfig) (string, error) {
	answer := ""
	if requirements != nil {
		answer = requirements.Cluster.ChartRepository
	}
	if answer == "" {
		answer = os.Getenv("JX_CHART_REPOSITORY")
	}
	if answer == "" {
		// assume default chart museum
		answer = "http://jenkins-x-chartmuseum:8080"
	}
	return answer, nil
}
