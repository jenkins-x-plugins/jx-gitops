package variablefinders

import (
	"os"

	jxc "github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-api/v3/pkg/config"
)

// FindRepositoryURL finds the chart repository URL via environment variables or the dev Environment CRD
func FindRepositoryURL(jxClient jxc.Interface, ns string, requirements *config.RequirementsConfig) (string, error) {
	answer := os.Getenv("JX_CHART_REPOSITORY")
	if answer == "" {
		if requirements != nil {
			answer = requirements.Cluster.ChartRepository
		}
	}
	if answer == "" {
		// assume default chart museum
		answer = "http://jenkins-x-chartmuseum:8080"
	}
	return answer, nil
}
