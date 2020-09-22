package variablefinders

import (
	jxc "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/pkg/errors"
)

// FindRequirements finds the requirements from the dev Environment CRD
func FindRequirements(jxClient jxc.Interface, ns string) (*config.RequirementsConfig, error) {
	// try the dev environment
	devEnv, err := jxenv.GetDevEnvironment(jxClient, ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find the dev Environment in namespace %s", ns)
	}

	requirements, err := config.GetRequirementsConfigFromTeamSettings(&devEnv.Spec.TeamSettings)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load requirements from dev environment")
	}
	return requirements, nil
}
