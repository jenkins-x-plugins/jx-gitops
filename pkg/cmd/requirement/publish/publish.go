package publish

import (
	"fmt"
	"path/filepath"

	v1 "github.com/jenkins-x/jx-api/v3/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/v3/pkg/config"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Publishes the current jx-requirements.yml to the dev Environment so it can be easily used in pipelines
`)

	cmdExample = templates.Examples(`
		%s requirements publish 
	`)
)

// Options the options for the command
type Options struct {
	Dir                  string
	EnvFile              string
	Namespace            string
	requirements         *config.RequirementsConfig
	requirementsFileName string
}

// NewCmdRequirementsPublish creates a command object for the command
func NewCmdRequirementsPublish() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "publish",
		Short:   "Publishes the current jx-requirements.yml to the dev Environment so it can be easily used in pipelines",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to run the git push command from")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace used to find the git operator secret for the git repository if running in cluster. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.EnvFile, "env-file", "", "", "the file name for the dev Environment. If not specified it defaults config-root/namespaces/jx/jxboot-helmfile-resources/dev-environment.yaml to within the directory")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	var err error
	o.requirements, o.requirementsFileName, err = config.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	if o.requirements == nil {
		return errors.Errorf("no 'jx-requirements.yml' file found in dir %s", o.Dir)
	}

	ns := o.requirements.Cluster.Namespace
	if ns == "" {
		ns = o.Namespace
		if ns == "" {
			ns, err = kubeclient.CurrentNamespace()
			if err != nil {
				return errors.Wrapf(err, "failed to find current namespace")
			}
		}
	}
	o.Namespace = ns

	if o.EnvFile == "" {
		o.EnvFile = filepath.Join(o.Dir, "config-root", "namespaces", ns, "jxboot-helmfile-resources", "dev-environment.yaml")
	}

	exists, err := files.FileExists(o.EnvFile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", o.EnvFile)
	}

	if !exists {
		return errors.Errorf("file does not exist %s", o.EnvFile)
	}

	env := &v1.Environment{}
	err = yamls.LoadFile(o.EnvFile, env)
	if err != nil {
		return errors.Wrapf(err, "failed to parse YAML file %s", o.EnvFile)
	}

	// lets replace the boot YAML
	data, err := yaml.Marshal(o.requirements)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal requirements to YAML from file %s", o.requirementsFileName)
	}

	env.Spec.TeamSettings.BootRequirements = string(data)
	err = yamls.SaveFile(env, o.EnvFile)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.EnvFile)
	}

	log.Logger().Infof("saved dev Environment file %s", info(o.EnvFile))
	return nil
}
