package variablefinders

import (
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/pkg/errors"
)

// FindRequirements finds the requirements from the dev Environment CRD
func FindRequirements(jxClient jxc.Interface, ns string) (*jxcore.RequirementsConfig, error) {
	// try the dev environment
	devEnv, err := jxenv.GetDevEnvironment(jxClient, ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find the dev Environment in namespace %s", ns)
	}

	requirements, err := jxcore.GetRequirementsConfigFromTeamSettings(&devEnv.Spec.TeamSettings)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load requirements from dev environment")
	}
	return requirements, nil
}
