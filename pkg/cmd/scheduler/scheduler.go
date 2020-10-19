package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx-api/v3/pkg/apis/jenkins.io/v1"
	jxconfig "github.com/jenkins-x/jx-api/v3/pkg/config"

	"github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-gitops/pkg/schedulerapi"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jenkins-x/jx-gitops/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	gyaml "github.com/ghodss/yaml"
)

const (
	// ConfigMapConfigFileName file name of the ConfigMap for the 'config' configuration
	ConfigMapConfigFileName = "config-cm.yaml"

	// ConfigMapPluginsFileName file name of the ConfigMap for the 'plugins' configuration
	ConfigMapPluginsFileName = "plugins-cm.yaml"

	// ConfigKey the name of key in the ConfigMap for the configuration of the `config`
	ConfigKey = "config.yaml"

	// PluginsKey the name of key in the ConfigMap for the configuration of the `plugins`
	PluginsKey = "plugins.yaml"
)

var (
	cmdLong = templates.LongDesc(`
		Generates the Lighthouse configuration from the SourceRepository and Scheduler resources
`)

	cmdExample = templates.Examples(`
		# regenerate the lighthouse configuration from the Environment, Scheduler, SourceRepository resources
		%s scheduler --dir config-root/namespaces/jx -out src/base/namespaces/jx/lighthouse-config

	`)

	sourceResourceFilter = kyamls.Filter{
		Kinds: []string{"jenkins.io/v1/Environment", "jenkins.io/v1/SourceRepository"},
	}

	schedulerResourceFilter = kyamls.Filter{
		Kinds: []string{"Scheduler"},
	}
)

// LabelOptions the options for the command
type Options struct {
	Dir           string
	OutDir        string
	SourceRepoDir string
	SchedulerDir  []string
	Namespace     string
	InRepoConfig  bool
}

// NewCmdScheduler creates a command object for the command
func NewCmdScheduler() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "scheduler",
		Aliases: []string{"schedulers", "lighthouse"},
		Short:   "Generates the Lighthouse configuration from the SourceRepository and Scheduler resources",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the current working directory")
	cmd.Flags().StringVarP(&o.SourceRepoDir, "repo-dir", "", "", "the directory to look for SourceRepository resources. If not specified defaults config-root/namespaces/$ns")
	cmd.Flags().StringArrayVarP(&o.SchedulerDir, "scheduler-dir", "", nil, "the directory to look for Scheduler resources. If not specified defaults 'schedulers' and 'versionStream/schedulers'")
	cmd.Flags().StringVarP(&o.OutDir, "out", "o", "", "the output directory for the generated config files. If not specified defaults to config-root/namespaces/$ns/lighthouse-config")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "jx", "the namespace for the SourceRepository and Scheduler resources")
	cmd.Flags().BoolVarP(&o.InRepoConfig, "in-repo-config", "", false, "enables in repo configuration in lighthouse")
	return cmd, o
}

func (o *Options) Run() error {
	ns := o.Namespace
	if ns == "" {
		ns = "jx"
	}
	if o.SourceRepoDir == "" {
		o.SourceRepoDir = filepath.Join(o.Dir, "config-root", "namespaces", ns)
	}
	if len(o.SchedulerDir) == 0 {
		paths := []string{
			"schedulers",
			filepath.Join(o.Dir, "versionStream", "schedulers"),
		}
		for _, path := range paths {
			exists, err := files.DirExists(path)
			if err != nil {
				return errors.Wrapf(err, "failed to check if path exists %s", path)
			}
			if exists {
				o.SchedulerDir = append(o.SchedulerDir, path)
			}
		}
	}
	if o.OutDir == "" {
		o.OutDir = filepath.Join(o.Dir, "config-root", "namespaces", ns, "lighthouse-config")
	}
	err := os.MkdirAll(o.OutDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create the output directory %s", o.OutDir)
	}

	var devEnv *v1.Environment
	var resources []runtime.Object

	schedulerMap := map[string]*schedulerapi.Scheduler{}
	repoListGroup := &v1.SourceRepositoryGroupList{}
	repoList := &v1.SourceRepositoryList{}

	sourceModifyFn := func(node *yaml.RNode, path string) (bool, error) {
		namespace := kyamls.GetNamespace(node, path)
		kind := kyamls.GetKind(node, path)
		name := kyamls.GetName(node, path)
		loaded := false
		switch kind {
		case "Environment":
			if name == "dev" {
				devEnv = &v1.Environment{}
				err = yamls.LoadFile(path, devEnv)
				if err != nil {
					return false, errors.Wrapf(err, "failed to load file %s", path)
				}
				loaded = true
			}

		case "SourceRepository":
			sr := &v1.SourceRepository{}
			err = yamls.LoadFile(path, sr)
			if err != nil {
				return false, errors.Wrapf(err, "failed to load file %s", path)
			}
			repoList.Items = append(repoList.Items, *sr)
			resources = append(resources, sr)
			loaded = true

		default:
			log.Logger().Infof("ignored %s name %s in namespace %s", kind, name, namespace)
		}
		if loaded {
			log.Logger().Infof("loaded %s name %s in namespace %s", kind, name, namespace)
		}
		return false, nil
	}
	err = kyamls.ModifyFiles(o.SourceRepoDir, sourceModifyFn, sourceResourceFilter)
	if err != nil {
		return errors.Wrapf(err, "failed to load resources from dir %s", o.SourceRepoDir)
	}

	log.Logger().Infof("loaded %d SourceRepository resources from %s", len(repoList.Items), o.SourceRepoDir)

	schedulerModifyFn := func(node *yaml.RNode, path string) (bool, error) {
		namespace := kyamls.GetNamespace(node, path)
		kind := kyamls.GetKind(node, path)
		name := kyamls.GetName(node, path)
		loaded := false
		scheduler := &schedulerapi.Scheduler{}
		err = yamls.LoadFile(path, scheduler)
		if err != nil {
			return false, errors.Wrapf(err, "failed to load file %s", path)
		}
		schedulerMap[name] = scheduler
		loaded = true
		if loaded {
			log.Logger().Infof("loaded %s name %s in namespace %s", kind, name, namespace)
		}
		return false, nil
	}
	for _, scheduleDir := range o.SchedulerDir {
		err = kyamls.ModifyFiles(scheduleDir, schedulerModifyFn, schedulerResourceFilter)
		if err != nil {
			return errors.Wrapf(err, "failed to load resources from dir %s", scheduleDir)
		}
	}
	log.Logger().Infof("loaded %d Scheduler resources from dirs %s", len(schedulerMap), strings.Join(o.SchedulerDir, ", "))

	if devEnv == nil {
		devEnv = &v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dev",
				Namespace: ns,
			},
			Spec: v1.EnvironmentSpec{
				Label:     "Dev",
				Namespace: ns,
			},
		}
	}
	teamSettings := &devEnv.Spec.TeamSettings

	resources = append(resources, devEnv)
	jxClient := fake.NewSimpleClientset(resources...)

	loadSchedulers := func(jxClient versioned.Interface, ns string) (map[string]*schedulerapi.Scheduler, *v1.SourceRepositoryGroupList, *v1.SourceRepositoryList, error) {
		return schedulerMap, repoListGroup, repoList, nil
	}

	config, plugins, err := pipelinescheduler.GenerateProw(true, true, jxClient, ns, teamSettings.DefaultScheduler.Name, devEnv, loadSchedulers)
	if err != nil {
		return errors.Wrapf(err, "failed to generate lighthouse configuration")
	}

	// lets check for in repo config
	flag := true
	for _, sr := range repoList.Items {
		if sr.Spec.Scheduler.Name == "in-repo" {
			if config.ProwConfig.InRepoConfig.Enabled == nil {
				config.ProwConfig.InRepoConfig.Enabled = map[string]*bool{}
			}
			fullName := scm.Join(sr.Spec.Org, sr.Spec.Repo)
			config.ProwConfig.InRepoConfig.Enabled[fullName] = &flag
		}
	}

	// lets process any templated values
	templater, err := o.createTemplater()
	if err != nil {
		return errors.Wrapf(err, "failed to create a templater")
	}
	config.Keeper.TargetURL, err = templater(config.Keeper.TargetURL)
	if err != nil {
		return errors.Wrapf(err, "failed to template the config.Keeper.TargetURL")
	}
	config.Keeper.PRStatusBaseURL, err = templater(config.Keeper.PRStatusBaseURL)
	if err != nil {
		return errors.Wrapf(err, "failed to template the config.Keeper.PRStatusBaseURL")
	}

	configConfigMap, err := createConfigMap(config, ns, "config", ConfigKey)
	if err != nil {
		return err
	}

	pluginsConfigMap, err := createConfigMap(plugins, ns, "plugins", PluginsKey)
	if err != nil {
		return err
	}

	// now lets save the files
	configFileName := filepath.Join(o.OutDir, ConfigMapConfigFileName)
	pluginsFileName := filepath.Join(o.OutDir, ConfigMapPluginsFileName)
	err = yamls.SaveFile(configConfigMap, configFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", configFileName)
	}
	err = yamls.SaveFile(pluginsConfigMap, pluginsFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", pluginsFileName)
	}
	log.Logger().Infof("generated config ConfigMap %s and plugins ConfigMap %s", termcolor.ColorInfo(configFileName), termcolor.ColorInfo(pluginsFileName))
	return nil
}

func (o *Options) createTemplater() (func(string) (string, error), error) {
	requirements, _, err := jxconfig.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	return func(templateText string) (string, error) {
		return EvaluateTemplate(templateText, requirements)
	}, nil
}

func createConfigMap(resource interface{}, ns string, name string, key string) (*corev1.ConfigMap, error) {
	data, err := gyaml.Marshal(resource)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal resource to YAML")
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string]string{
			key: string(data),
		},
	}, nil
}
