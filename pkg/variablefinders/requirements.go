package variablefinders

import (
	"github.com/imdario/mergo"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/requirements"
	"github.com/pkg/errors"
)

// FindRequirements finds the requirements from the dev Environment CRD
func FindRequirements(g gitclient.Interface, jxClient jxc.Interface, ns string, dir string) (*jxcore.RequirementsConfig, error) {
	settings, err := requirements.LoadSettings(dir, true)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load settings")
	}

	if settings == nil {
		req, err := requirements.GetClusterRequirementsConfig(g, jxClient)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load requirements from dev environment")
		}
		return req, nil
	}

	gitURL := settings.Spec.GitURL
	if gitURL == "" {
		if ns == "" {
			ns = jxcore.DefaultNamespace
		}
		env, err := jxenv.GetDevEnvironment(jxClient, ns)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get dev environment")
		}
		if env == nil {
			return nil, errors.Errorf("failed to find a dev environment source url as there is no 'dev' Environment resource in namespace %s", ns)
		}
		gitURL = env.Spec.Source.URL
		if gitURL == "" {
			return nil, errors.New("failed to find a dev environment source url on development environment resource")
		}
	}

	// now lets merge the local requirements with the dev environment so that we can locally override things
	// while inheriting common stuff
	req, err := requirements.GetRequirementsFromGit(g, gitURL)
	ss := &settings.Spec

	if ss.Destination != nil {
		err = mergo.Merge(&req.Cluster.DestinationConfig, ss.Destination, mergo.WithOverride)
		if err != nil {
			return nil, errors.Wrap(err, "error merging requirements.spec.cluster Destination from settings")
		}
	}

	// merge the environments now
	if ss.IgnoreDevEnvironments {
		req.Environments = ss.PromoteEnvironments
	} else {
		for i := range ss.PromoteEnvironments {
			env := &ss.PromoteEnvironments[i]

			found := false
			key := env.Key
			for j := range req.Environments {
				sharedEnv := &req.Environments[j]
				if key == sharedEnv.Key {
					found = true
					err = mergo.Merge(sharedEnv, env)
					if err != nil {
						return nil, errors.Wrapf(err, "error merging requirements.environment for %s,", key)
					}
				}
			}
			if !found {
				req.Environments = append(req.Environments, *env)
			}
		}
	}
	return req, nil
}
