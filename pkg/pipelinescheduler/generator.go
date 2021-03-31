package pipelinescheduler

import (
	"context"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/schedulerapi"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/lighthouse-client/pkg/config"
	"github.com/jenkins-x/lighthouse-client/pkg/plugins"
	v1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/ghodss/yaml"

	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateProw will generate the prow config for the namespace
func GenerateProw(gitOps bool, autoApplyConfigUpdater bool, jxClient versioned.Interface, namespace string, teamSchedulerName string, devEnv *jenkinsv1.Environment, loadSchedulerResourcesFunc func(versioned.Interface, string) (map[string]*schedulerapi.Scheduler, *jenkinsv1.SourceRepositoryList, error)) (*config.Config,
	*plugins.Configuration, error) {
	schedulers, sourceRepos, err := loadSchedulerResourcesFunc(jxClient, namespace)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "loading scheduler resources")
	}
	if sourceRepos == nil || len(sourceRepos.Items) < 1 {
		return nil, nil, errors.New("No source repository resources were found")
	}
	defaultScheduler := schedulers[teamSchedulerName]
	leaves := make([]*SchedulerLeaf, 0)
	for _, sourceRepo := range sourceRepos.Items {
		applicableSchedulers := []*schedulerapi.SchedulerSpec{}
		// Apply config-updater to devEnv
		applicableSchedulers = addConfigUpdaterToDevEnv(gitOps, autoApplyConfigUpdater, applicableSchedulers, devEnv, &sourceRepo.Spec)
		// Apply repo scheduler
		applicableSchedulers = addRepositoryScheduler(sourceRepo, schedulers, applicableSchedulers)
		// Apply team scheduler
		applicableSchedulers = addTeamScheduler(teamSchedulerName, defaultScheduler, applicableSchedulers)
		if len(applicableSchedulers) < 1 {
			continue
		}
		merged, err := Build(applicableSchedulers)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building scheduler")
		}
		leaves = append(leaves, &SchedulerLeaf{
			Repo:          sourceRepo.Spec.Repo,
			Org:           sourceRepo.Spec.Org,
			SchedulerSpec: merged,
		})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building prow config")
		}
	}
	cfg, plugs, err := BuildProwConfig(leaves)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "building prow config")
	}
	if cfg != nil {
		cfg.PodNamespace = namespace
		cfg.LighthouseJobNamespace = namespace
	}
	return cfg, plugs, nil
}

func addTeamScheduler(defaultSchedulerName string, defaultScheduler *schedulerapi.Scheduler, applicableSchedulers []*schedulerapi.SchedulerSpec) []*schedulerapi.SchedulerSpec {
	if defaultScheduler != nil && len(applicableSchedulers) == 0 {
		applicableSchedulers = append([]*schedulerapi.SchedulerSpec{&defaultScheduler.Spec}, applicableSchedulers...)
	} else {
		if defaultSchedulerName != "" {
			log.Logger().Debugf("A team pipeline scheduler named %s was configured but could not be found", defaultSchedulerName)
		}
	}
	return applicableSchedulers
}

func addRepositoryScheduler(sourceRepo jenkinsv1.SourceRepository, lookup map[string]*schedulerapi.Scheduler, applicableSchedulers []*schedulerapi.SchedulerSpec) []*schedulerapi.SchedulerSpec {
	if sourceRepo.Spec.Scheduler.Name != "" {
		scheduler := lookup[sourceRepo.Spec.Scheduler.Name]
		if scheduler != nil {
			applicableSchedulers = append([]*schedulerapi.SchedulerSpec{&scheduler.Spec}, applicableSchedulers...)
		} else {
			log.Logger().Warnf("A scheduler named %s is referenced by repository(%s) but could not be found", sourceRepo.Spec.Scheduler.Name, sourceRepo.Name)
		}
	}
	return applicableSchedulers
}

func addConfigUpdaterToDevEnv(gitOps bool, autoApplyConfigUpdater bool, applicableSchedulers []*schedulerapi.SchedulerSpec, devEnv *jenkinsv1.Environment, sourceRepo *jenkinsv1.SourceRepositorySpec) []*schedulerapi.SchedulerSpec {
	if gitOps && autoApplyConfigUpdater && strings.Contains(devEnv.Spec.Source.URL, sourceRepo.Org+"/"+sourceRepo.Repo) {
		maps := make(map[string]schedulerapi.ConfigMapSpec)
		maps["env/prow/job.yaml"] = schedulerapi.ConfigMapSpec{
			Name: "config",
		}
		maps["env/prow/plugins.yaml"] = schedulerapi.ConfigMapSpec{
			Name: "plugins",
		}
		environmentUpdaterSpec := &schedulerapi.SchedulerSpec{
			ConfigUpdater: &schedulerapi.ConfigUpdater{
				Map: maps,
			},
			Plugins: &schedulerapi.ReplaceableSliceOfStrings{
				Items: []string{"config-updater"},
			},
		}
		applicableSchedulers = append([]*schedulerapi.SchedulerSpec{environmentUpdaterSpec}, applicableSchedulers...)
	}
	return applicableSchedulers
}

//ApplyDirectly directly applies the prow config to the cluster
func ApplyDirectly(kubeClient kubernetes.Interface, namespace string, cfg *config.Config,
	plugs *plugins.Configuration) error {
	cfgYaml, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrapf(err, "marshalling config to yaml")
	}
	cfgConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"job.yaml": string(cfgYaml),
		},
	}
	plugsYaml, err := yaml.Marshal(plugs)
	if err != nil {
		return errors.Wrapf(err, "marshalling plugins to yaml")
	}
	plugsConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "plugins",
			Namespace: namespace,
		},
		Data: map[string]string{
			"plugins.yaml": string(plugsYaml),
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), cfgConfigMap, metav1.UpdateOptions{})
	if kubeerrors.IsNotFound(err) {
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cfgConfigMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "creating ConfigMap config")
		}
	} else if err != nil {
		return errors.Wrapf(err, "updating ConfigMap config")
	}
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), plugsConfigMap, metav1.UpdateOptions{})
	if kubeerrors.IsNotFound(err) {
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), plugsConfigMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "creating ConfigMap plugins")
		}
	} else if err != nil {
		return errors.Wrapf(err, "updating ConfigMap plugins")
	}
	return nil
}
