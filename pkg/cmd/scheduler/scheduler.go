package scheduler

import (
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/jenkins-x/jx-logging/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jenkins-x/jx-gitops/pkg/kyamls"
	"github.com/jenkins-x/jx-gitops/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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
		Converts all Secret resources in the path to ExternalSecret CRDs
`)

	cmdExample = templates.Examples(`
		# updates recursively labels all resources in the current directory 
		%s scheduler --dir=.
	`)

	schedulerResourceFilter = kyamls.Filter{
		Kinds: []string{"jenkins.io/v1/Environment", "jenkins.io/v1/Scheduler", "jenkins.io/v1/SourceRepository"},
	}
)

// LabelOptions the options for the command
type Options struct {
	Dir       string
	OutDir    string
	Namespace string
}

// NewCmdScheduler creates a command object for the command
func NewCmdScheduler() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "scheduler",
		Aliases: []string{"schedulers", "extsec"},
		Short:   "Generates the Lighthouse configuration from the SourceRepository and Scheduler resources",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.OutDir, "out", "o", "", "the output directory for the generated config files")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "jx", "the namespace for the SourceRepository and Scheduler resources")
	return cmd, o
}

func (o *Options) Run() error {
	dir := o.Dir
	ns := o.Namespace
	if ns == "" {
		ns = "jx"
	}
	if o.OutDir == "" {
		o.OutDir = filepath.Join(o.Dir, "src", "base", "namespaces", ns, "lighthouse-config")
	}
	err := os.MkdirAll(o.OutDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create the output directory %s", o.OutDir)
	}

	var devEnv *v1.Environment
	var resources []runtime.Object

	schedulerMap := map[string]*v1.Scheduler{}
	repoListGroup := &v1.SourceRepositoryGroupList{}
	repoList := &v1.SourceRepositoryList{}

	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
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

		case "Scheduler":
			scheduler := &v1.Scheduler{}
			err = yamls.LoadFile(path, scheduler)
			if err != nil {
				return false, errors.Wrapf(err, "failed to load file %s", path)
			}
			schedulerMap[name] = scheduler
			resources = append(resources, scheduler)
			loaded = true

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

	err = kyamls.ModifyFiles(dir, modifyFn, schedulerResourceFilter)
	if err != nil {
		return errors.Wrapf(err, "failed to load resources from dir %s", dir)
	}

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

	loadSchedulers := func(jxClient versioned.Interface, ns string) (map[string]*v1.Scheduler, *v1.SourceRepositoryGroupList, *v1.SourceRepositoryList, error) {
		return schedulerMap, repoListGroup, repoList, nil
	}

	config, plugins, err := pipelinescheduler.GenerateProw(true, true, jxClient, ns, teamSettings.DefaultScheduler.Name, devEnv, loadSchedulers)
	if err != nil {
		return errors.Wrapf(err, "failed to generate lighthouse configuration")
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

func createConfigMap(resource interface{}, ns string, name string, key string) (*corev1.ConfigMap, error) {
	data, err := yaml.Marshal(resource)
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
