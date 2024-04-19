package create

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Creates any missing SourceRepository resources
`)

	cmdExample = templates.Examples(`
		# creates any missing SourceRepository resources  
		%s repository create https://github.com/myorg/myrepo.git
	`)
)

// LabelOptions the options for the command
type Options struct {
	kyamls.Filter
	Dir        string
	SourceDir  string
	ConfigFile string
}

// NewCmdCreateRepository creates a command object for the command
func NewCmdCreateRepository() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Creates any missing SourceRepository resources",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory look for the 'jx-requirements.yml` file")
	cmd.Flags().StringVarP(&o.SourceDir, "source-dir", "s", "", "the directory to look for and generate the SourceConfig files")
	cmd.Flags().StringVarP(&o.ConfigFile, "config", "c", "", "the configuration file to load for the repository configurations. If not specified we look in ./.jx/gitops/source-config.yaml")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run transforms the YAML files
func (o *Options) Run() error {
	if o.SourceDir == "" {
		o.SourceDir = filepath.Join(o.Dir, "config-root", "namespaces", "jx", "source-repositories")
	}

	if o.ConfigFile == "" {
		o.ConfigFile = filepath.Join(o.Dir, ".jx", "gitops", v1alpha1.SourceConfigFileName)
	}

	exists, err := files.FileExists(o.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", o.ConfigFile)
	}
	if !exists {
		log.Logger().Infof("file does not exist: %s so not defaulting any SourceConfig resources", o.ConfigFile)
		return nil
	}

	err = os.MkdirAll(o.SourceDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", o.SourceDir)
	}

	config := &v1alpha1.SourceConfig{}

	err = yamls.LoadFile(o.ConfigFile, config)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", o.ConfigFile)
	}

	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		for j := range group.Repositories {
			repo := &group.Repositories[j]
			err = o.ensureSourceRepositoryExists(config, group, repo)
			if err != nil {
				return errors.Wrapf(err, "failed to ensure repository is created for %s/%s", group.Owner, repo.Name)
			}
		}
	}
	return nil
}

func (o *Options) ensureSourceRepositoryExists(config *v1alpha1.SourceConfig, group *v1alpha1.RepositoryGroup, repo *v1alpha1.Repository) error {
	err := sourceconfigs.DefaultValues(config, group, repo)
	if err != nil {
		return errors.Wrapf(err, "failed to default values")
	}

	owner := group.Owner
	repoName := repo.Name
	name := naming.ToValidName(fmt.Sprintf("%s-%s", owner, repoName))
	fileName := filepath.Join(o.SourceDir, name+".yaml")
	exists, err := files.FileExists(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to check for file %s", fileName)
	}

	sr := &v1.SourceRepository{}
	if exists {
		err = yamls.LoadFile(fileName, sr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse file %s", fileName)
		}
	}

	// lets make sure we are populated correctly
	modified := false
	if sr.APIVersion == "" {
		sr.APIVersion = "jenkins.io/v1"
		modified = true
	}
	if sr.Kind == "" {
		sr.Kind = "SourceRepository"
		modified = true
	}
	if sr.Name != name {
		sr.Name = name
		modified = true
	}
	s := &sr.Spec
	if s.Org != owner {
		s.Org = owner
		modified = true
	}
	if s.Repo != repoName {
		s.Repo = repoName
		modified = true
	}
	if sr.Labels == nil {
		sr.Labels = map[string]string{}
	}
	if sr.Labels["owner"] != owner {
		sr.Labels["owner"] = owner
		modified = true
	}
	if sr.Labels["repository"] != repoName {
		// Convert / to - for nested repositories, so that it's a valid kubernetes label value
		sr.Labels["repository"] = naming.ToValidName(repoName)
		modified = true
	}
	if group.ProviderKind != "" && sr.Labels["provider"] != owner {
		sr.Labels["provider"] = group.ProviderKind
		modified = true
	}
	if group.Provider != "" && s.Provider != group.Provider {
		s.Provider = group.Provider
		modified = true
	}
	if group.ProviderKind != "" && s.ProviderKind != group.ProviderKind {
		s.ProviderKind = group.ProviderKind
		modified = true
	}
	if group.ProviderName != "" && s.ProviderName != group.ProviderName {
		s.ProviderName = group.ProviderName
		modified = true
	}
	if repo.URL != "" && s.URL != repo.URL {
		s.URL = repo.URL
		modified = true
	}
	if repo.HTTPCloneURL != "" && s.HTTPCloneURL != repo.HTTPCloneURL {
		s.HTTPCloneURL = repo.HTTPCloneURL
		modified = true
	}
	if repo.SSHCloneURL != "" && s.SSHCloneURL != repo.SSHCloneURL {
		s.SSHCloneURL = repo.SSHCloneURL
		modified = true
	}
	if repo.Description != "" && s.Description != repo.Description {
		s.Description = repo.Description
		modified = true
	}
	if repo.Scheduler != "" && s.Scheduler.Name != repo.Scheduler {
		s.Scheduler.Name = repo.Scheduler
		modified = true
	}

	if !modified {
		return nil
	}
	err = yamls.SaveFile(sr, fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	action := "created"
	if exists {
		action = "modified"
	}
	log.Logger().Debugf("%s file %s", action, termcolor.ColorInfo(fileName))
	return nil
}
