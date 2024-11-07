package variablefinders

import (
	"github.com/imdario/mergo"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/requirements"
	"github.com/pkg/errors"
)

// FindRequirements finds the requirements from the dev Environment CRD
func FindRequirements(g gitclient.Interface, jxClient jxc.Interface, ns, dir, owner, repo string) (*jxcore.RequirementsConfig, error) {
	// now lets merge the local requirements with the dev environment so that we can locally override things
	// while inheriting common stuff

	settings, clusterDir, err := GetSettings(g, jxClient, ns, dir, owner, repo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get settings")
	}

	requirementsConfig, _, err := jxcore.LoadRequirementsConfig(clusterDir, false)
	var req *jxcore.RequirementsConfig
	if err != nil || requirementsConfig == nil {
		r := jxcore.NewRequirementsConfig()
		req = &r.Spec
	} else {
		req = &requirementsConfig.Spec
	}

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
					err = mergo.Merge(sharedEnv, env, mergo.WithOverride)
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

// GetSettings mergers and returns the settings from .jx/gitops/source-config.yaml in the cluster repo and .jx/settings.yaml in the current directory
func GetSettings(g gitclient.Interface, jxClient jxc.Interface, ns, dir, owner, repo string) (*jxcore.Settings, string, error) {
	settings, err := requirements.LoadSettings(dir, true)
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to load settings")
	}
	if settings == nil {
		// lets use an empty settings file
		settings = &jxcore.Settings{}
	}
	gitURL := ""
	if settings != nil {
		gitURL = settings.Spec.GitURL
	}
	if gitURL == "" {
		if ns == "" {
			ns = jxcore.DefaultNamespace
		}
		env, err := jxenv.GetDevEnvironment(jxClient, ns)
		if err != nil {
			return nil, "", errors.Wrap(err, "failed to get dev environment")
		}
		if env == nil {
			return nil, "", errors.Errorf("failed to find a dev environment source url as there is no 'dev' Environment resource in namespace %s", ns)
		}
		gitURL = env.Spec.Source.URL
		if gitURL == "" {
			return nil, "", errors.New("failed to find a dev environment source url on development environment resource")
		}
	}
	combinedConfigFiles := []string{"/" + jxcore.RequirementsConfigFileName, sourceconfigs.SourceConfigFile}
	if len(combinedConfigFiles) == 0 {
		return nil, "", errors.New("combined config files are empty")
	}
	clusterDir, err := requirements.PartialCloneClusterRepo(g, gitURL, true, combinedConfigFiles...)
	if err != nil {
		return nil, "", err
	}

	// lets see if we have organisation settings
	srcConfig, err := sourceconfigs.LoadSourceConfig(clusterDir, true)
	if err != nil {
		return nil, clusterDir, errors.Wrapf(err, "failed to load source configs")
	}
	groupSettings := sourceconfigs.FindSettings(srcConfig, owner, repo)

	settings, err = mergeSettings(settings, groupSettings)
	if err != nil {
		return nil, clusterDir, errors.Wrapf(err, "failed to merge settings")
	}
	return settings, clusterDir, nil
}

// mergeSettings merges the local and group settings
func mergeSettings(local *jxcore.Settings, groupConfig *jxcore.SettingsConfig) (*jxcore.Settings, error) {
	var group *jxcore.Settings
	if groupConfig != nil {
		group = &jxcore.Settings{
			Spec: *groupConfig,
		}
	}
	if local == nil {
		return group, nil
	}
	if group == nil {
		return local, nil
	}
	err := mergo.Merge(group, local, mergo.WithOverride)
	if err != nil {
		return nil, errors.Wrap(err, "error merging local and source config group Settings")
	}
	return group, nil
}
