package variablefinders

import (
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/requirements"
	"github.com/pkg/errors"
)

// FindRequirements finds the requirements from the dev Environment CRD
func FindRequirements(jxClient jxc.Interface, ns string, g gitclient.Interface) (*jxcore.RequirementsConfig, error) {
	req, err := requirements.GetClusterRequirementsConfig(g, jxClient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load requirements from dev environment")
	}
	return req, nil
}
