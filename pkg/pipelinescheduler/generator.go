package pipelinescheduler

import (
	"strings"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	v1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/ghodss/yaml"

	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateProw will generate the prow config for the namespace
func GenerateProw(gitOps bool, autoApplyConfigUpdater bool, jxClient versioned.Interface, namespace string, teamSchedulerName string, devEnv *jenkinsv1.Environment, loadSchedulerResourcesFunc func(versioned.Interface, string) (map[string]*jenkinsv1.Scheduler, *jenkinsv1.SourceRepositoryGroupList, *jenkinsv1.SourceRepositoryList, error)) (*config.Config,
	*plugins.Configuration, error) {
	if loadSchedulerResourcesFunc == nil {
		loadSchedulerResourcesFunc = loadSchedulerResources
	}
	schedulers, sourceRepoGroups, sourceRepos, err := loadSchedulerResourcesFunc(jxClient, namespace)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "loading scheduler resources")
	}
	if sourceRepos == nil || len(sourceRepos.Items) < 1 {
		return nil, nil, errors.New("No source repository resources were found")
	}
	defaultScheduler := schedulers[teamSchedulerName]
	leaves := make([]*SchedulerLeaf, 0)
	for _, sourceRepo := range sourceRepos.Items {
		applicableSchedulers := []*jenkinsv1.SchedulerSpec{}
		// Apply config-updater to devEnv
		applicableSchedulers = addConfigUpdaterToDevEnv(gitOps, autoApplyConfigUpdater, applicableSchedulers, devEnv, &sourceRepo.Spec)
		// Apply repo scheduler
		applicableSchedulers = addRepositoryScheduler(sourceRepo, schedulers, applicableSchedulers)
		// Apply project schedulers
		applicableSchedulers = addProjectSchedulers(sourceRepoGroups, sourceRepo, schedulers, applicableSchedulers)
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

func loadSchedulerResources(jxClient versioned.Interface, namespace string) (map[string]*jenkinsv1.Scheduler, *jenkinsv1.SourceRepositoryGroupList, *jenkinsv1.SourceRepositoryList, error) {
	schedulers, err := jxClient.JenkinsV1().Schedulers(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}
	if len(schedulers.Items) == 0 {
		return nil, nil, nil, errors.New("No pipeline schedulers are configured")
	}
	lookup := make(map[string]*jenkinsv1.Scheduler)
	for _, item := range schedulers.Items {
		lookup[item.Name] = item.DeepCopy()
	}
	// Process Schedulers linked to SourceRepositoryGroups
	sourceRepoGroups, err := jxClient.JenkinsV1().SourceRepositoryGroups(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "Error finding source repository groups")
	}
	// Process Schedulers linked to SourceRepositoryGroups
	sourceRepos, err := jxClient.JenkinsV1().SourceRepositories(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "Error finding source repositories")
	}
	return lookup, sourceRepoGroups, sourceRepos, nil
}

func addTeamScheduler(defaultSchedulerName string, defaultScheduler *jenkinsv1.Scheduler, applicableSchedulers []*jenkinsv1.SchedulerSpec) []*jenkinsv1.SchedulerSpec {
	if defaultScheduler != nil {
		applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{&defaultScheduler.Spec}, applicableSchedulers...)
	} else {
		if defaultSchedulerName != "" {
			log.Logger().Warnf("A team pipeline scheduler named %s was configured but could not be found", defaultSchedulerName)
		}
	}
	return applicableSchedulers
}

func addRepositoryScheduler(sourceRepo jenkinsv1.SourceRepository, lookup map[string]*jenkinsv1.Scheduler, applicableSchedulers []*jenkinsv1.SchedulerSpec) []*jenkinsv1.SchedulerSpec {
	if sourceRepo.Spec.Scheduler.Name != "" {
		scheduler := lookup[sourceRepo.Spec.Scheduler.Name]
		if scheduler != nil {
			applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{&scheduler.Spec}, applicableSchedulers...)
		} else {
			log.Logger().Warnf("A scheduler named %s is referenced by repository(%s) but could not be found", sourceRepo.Spec.Scheduler.Name, sourceRepo.Name)
		}
	}
	return applicableSchedulers
}

func addProjectSchedulers(sourceRepoGroups *jenkinsv1.SourceRepositoryGroupList, sourceRepo jenkinsv1.SourceRepository, lookup map[string]*jenkinsv1.Scheduler, applicableSchedulers []*jenkinsv1.SchedulerSpec) []*jenkinsv1.SchedulerSpec {
	if sourceRepoGroups != nil {
		for _, sourceGroup := range sourceRepoGroups.Items {
			for _, groupRepo := range sourceGroup.Spec.SourceRepositorySpec {
				if groupRepo.Name == sourceRepo.Name {
					if sourceGroup.Spec.Scheduler.Name != "" {
						scheduler := lookup[sourceGroup.Spec.Scheduler.Name]
						if scheduler != nil {
							applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{&scheduler.Spec}, applicableSchedulers...)
						} else {
							log.Logger().Warnf("A scheduler named %s is referenced by repository group(%s) but could not be found", sourceGroup.Spec.Scheduler.Name, sourceGroup.Name)
						}
					}
				}
			}
		}
	}
	return applicableSchedulers
}

func addConfigUpdaterToDevEnv(gitOps bool, autoApplyConfigUpdater bool, applicableSchedulers []*jenkinsv1.SchedulerSpec, devEnv *jenkinsv1.Environment, sourceRepo *jenkinsv1.SourceRepositorySpec) []*jenkinsv1.SchedulerSpec {
	if gitOps && autoApplyConfigUpdater && strings.Contains(devEnv.Spec.Source.URL, sourceRepo.Org+"/"+sourceRepo.Repo) {
		maps := make(map[string]jenkinsv1.ConfigMapSpec)
		maps["env/prow/job.yaml"] = jenkinsv1.ConfigMapSpec{
			Name: "config",
		}
		maps["env/prow/plugins.yaml"] = jenkinsv1.ConfigMapSpec{
			Name: "plugins",
		}
		environmentUpdaterSpec := &jenkinsv1.SchedulerSpec{
			ConfigUpdater: &jenkinsv1.ConfigUpdater{
				Map: maps,
			},
			Plugins: &jenkinsv1.ReplaceableSliceOfStrings{
				Items: []string{"config-updater"},
			},
		}
		applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{environmentUpdaterSpec}, applicableSchedulers...)
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
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(cfgConfigMap)
	if kubeerrors.IsNotFound(err) {
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(cfgConfigMap)
		if err != nil {
			return errors.Wrapf(err, "creating ConfigMap config")
		}
	} else if err != nil {
		return errors.Wrapf(err, "updating ConfigMap config")
	}
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(plugsConfigMap)
	if kubeerrors.IsNotFound(err) {
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(plugsConfigMap)
		if err != nil {
			return errors.Wrapf(err, "creating ConfigMap plugins")
		}
	} else if err != nil {
		return errors.Wrapf(err, "updating ConfigMap plugins")
	}
	return nil
}

//ApplySchedulersDirectly directly applies pipeline schedulers to the cluster
func ApplySchedulersDirectly(jxClient versioned.Interface, namespace string, sourceRepositoryGroups []*jenkinsv1.SourceRepositoryGroup, sourceRepositories []*jenkinsv1.SourceRepository, schedulers map[string]*jenkinsv1.Scheduler, devEnv *jenkinsv1.Environment) error {
	log.Logger().Infof("Applying scheduler configuration to namespace %s", namespace)
	err := jxClient.JenkinsV1().Schedulers(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error removing existing schedulers")
	}
	for _, scheduler := range schedulers {
		_, err := jxClient.JenkinsV1().Schedulers(namespace).Update(scheduler)
		if kubeerrors.IsNotFound(err) {
			_, err := jxClient.JenkinsV1().Schedulers(namespace).Create(scheduler)
			if err != nil {
				return errors.Wrapf(err, "creating scheduler")
			}
		} else if err != nil {
			return errors.Wrapf(err, "updating scheduler")
		}
		if scheduler.Name == "default-scheduler" {
			devEnv.Spec.TeamSettings.DefaultScheduler.Name = scheduler.Name
			devEnv.Spec.TeamSettings.DefaultScheduler.Kind = "Scheduler"
			_, err = jxClient.JenkinsV1().Environments(namespace).PatchUpdate(devEnv)
			if err != nil {
				return errors.Wrapf(err, "patch updating env %v", devEnv)
			}
		}
	}
	for _, repo := range sourceRepositories {
		sourceRepo, err := GetOrCreateSourceRepository(jxClient, namespace, repo.Spec.Repo, repo.Spec.Org, repo.Spec.Provider)
		if err != nil || sourceRepo == nil {
			return errors.New("Getting / creating source repo")
		}
		sourceRepo.Spec.Scheduler.Name = repo.Spec.Scheduler.Name
		sourceRepo.Spec.Scheduler.Kind = repo.Spec.Scheduler.Kind
		_, err = jxClient.JenkinsV1().SourceRepositories(namespace).Update(sourceRepo)
		if err != nil {
			return errors.Wrapf(err, "updating source repo")
		}
	}
	for _, repoGroup := range sourceRepositoryGroups {
		_, err := jxClient.JenkinsV1().SourceRepositoryGroups(namespace).Update(repoGroup)
		if kubeerrors.IsNotFound(err) {
			_, err := jxClient.JenkinsV1().SourceRepositoryGroups(namespace).Create(repoGroup)
			if err != nil {
				return errors.Wrapf(err, "creating source repo group")
			}
		} else if err != nil {
			return errors.Wrapf(err, "updating source repo group")
		}
	}

	return nil
}
