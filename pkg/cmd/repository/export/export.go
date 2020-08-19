package export

import (
	"fmt"
	"os"
	"path/filepath"

	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cmdLong = templates.LongDesc(`
		"Exports the 'source-config.yaml' file from the kubernetes resources in the current cluster
`)

	cmdExample = templates.Examples(`
		# creates/populates the .jx/gitops/source-config.yaml file with any SourceRepository resources in the current cluster
		%s repository export
	`)
)

// LabelOptions the options for the command
type Options struct {
	kyamls.Filter
	Dir        string
	ConfigFile string
	Namespace  string
	JXClient   versioned.Interface
}

// NewCmdExportConfig creates a command object for the command
func NewCmdExportConfig() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Exports the 'source-config.yaml' file from the kubernetes resources in the current cluster",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory look for the 'jx-requirements.yml` file")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "the namespace to look for SourceRepository, SourceRepositoryGroup and Scheduler resources")
	cmd.Flags().StringVarP(&o.ConfigFile, "config", "c", "", "the configuration file to load for the repository configurations. If not specified we look in ./.jx/gitops/source-repositories.yaml")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run transforms the YAML files
func (o *Options) Run() error {
	if o.ConfigFile == "" {
		o.ConfigFile = filepath.Join(o.Dir, ".jx", "gitops", v1alpha1.SourceConfigFileName)
	}

	exists, err := files.FileExists(o.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", o.ConfigFile)
	}

	config := &v1alpha1.SourceConfig{}
	if exists {
		err = yamls.LoadFile(o.ConfigFile, config)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", o.ConfigFile)
		}
	}
	if config.APIVersion == "" {
		config.APIVersion = v1alpha1.APIVersion
	}
	if config.Kind == "" {
		config.Kind = v1alpha1.KindSourceConfig
	}

	dir := filepath.Dir(o.ConfigFile)
	err = os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", dir)
	}

	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}

	ns := o.Namespace
	srList, err := o.JXClient.JenkinsV1().SourceRepositories(ns).List(metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to load SourceRepositories in namespace %s", ns)
	}

	err = o.populateConfig(config, srList)
	if err != nil {
		return errors.Wrapf(err, "failed to populate config")
	}

	err = yamls.SaveFile(config, o.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "failed to save config file %s", o.ConfigFile)
	}

	log.Logger().Infof("modified file %s", termcolor.ColorInfo(o.ConfigFile))
	return nil
}

func (o *Options) populateConfig(config *v1alpha1.SourceConfig, srList *jenkinsv1.SourceRepositoryList) error {
	if srList != nil {
		for i := range srList.Items {
			sr := &srList.Items[i]
			owner := sr.Spec.Org
			if owner == "" {
				log.Logger().Warnf("ignoring SourceRepository %s with no owner", sr.Name)
				continue
			}
			repoName := sr.Spec.Repo
			if repoName == "" {
				log.Logger().Warnf("ignoring SourceRepository %s with no repo", sr.Name)
				continue
			}
			group := sourceconfigs.GetOrCreateGroup(config, owner)
			repo := sourceconfigs.GetOrCreateRepository(group, repoName)

			err := sourceconfigs.DefaultValues(config, group, repo)
			if err != nil {
				return errors.Wrapf(err, "failed to default values")
			}

			s := &sr.Spec
			if repo.Description == "" {
				repo.Description = s.Description
			}
			if s.URL != "" {
				repo.URL = s.URL
			}
			if s.HTTPCloneURL != "" {
				repo.HTTPCloneURL = s.HTTPCloneURL
			}
			if s.SSHCloneURL != "" {
				repo.SSHCloneURL = s.SSHCloneURL
			}
			if s.Scheduler.Name != "" {
				repo.Scheduler = s.Scheduler.Name
			}
		}
	}
	return nil
}
