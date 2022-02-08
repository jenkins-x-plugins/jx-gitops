package variablefinders

import (
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
)

// DockerRegistryOrg returns the docker registry org for the given requirements and git owner
func DockerRegistryOrg(requirements *jxcore.RequirementsConfig, owner string) (string, error) {
	answer := ""
	if requirements != nil {
		answer = requirements.Cluster.DockerRegistryOrg
		if answer == "" && requirements.Cluster.Provider == "gke" {
			answer = requirements.Cluster.ProjectID
		}
	}
	if answer == "" {
		answer = naming.ToValidName(owner)
	}
	return answer, nil
}
