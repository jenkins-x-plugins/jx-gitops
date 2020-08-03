package merge

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Merges values from the given file to the local jx-requirements.yml file

This lets you take requirements from, say, the output of a terraform plan and merge with any other changes inside your GitOps repository
`)

	cmdExample = templates.Examples(`
		%s requirements merge -f /tmp/jx-requirements.yml 
	`)
)

// Options the options for the command
type Options struct {
	Dir                  string
	File                 string
	requirements         *config.RequirementsConfig
	requirementsFileName string
}

// NewCmdRequirementsResolve creates a command object for the command
func NewCmdRequirementsMerge() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "merge",
		Short:   "Merges values from the given file to the local jx-requirements.yml file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the source directory to merge changes into")
	cmd.Flags().StringVarP(&o.File, "file", "f", "", "the requirements file to merge into the source directory")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.File == "" {
		return options.MissingOption("file")
	}
	var err error
	o.requirements, o.requirementsFileName, err = config.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	if o.requirementsFileName == "" {
		o.requirementsFileName = filepath.Join(o.Dir, config.RequirementsConfigFileName)
	}

	requirementChanges, err := config.LoadRequirementsConfigFile(o.File, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load changes from file: %s", o.File)
	}
	if requirementChanges == nil {
		return errors.Errorf("no requirements config found for file: %s", o.File)
	}

	exists := false
	if o.requirements != nil {
		exists, err = files.FileExists(o.requirementsFileName)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", o.requirementsFileName)
		}
	}

	if exists {
		err = o.MergeChanges(requirementChanges)
		if err != nil {
			return errors.Wrapf(err, "failed to merge changes from %s", o.File)
		}
	} else {
		o.requirements = requirementChanges
	}

	err = o.requirements.SaveConfig(o.requirementsFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.requirementsFileName)
	}
	log.Logger().Infof("saved file %s", termcolor.ColorInfo(o.requirementsFileName))
	return nil

}

// MergeChanges merges changes from the given requirements into the source
func (o *Options) MergeChanges(changes *config.RequirementsConfig) error {
	to := o.requirements
	cluster := changes.Cluster

	// lets pull in any values missing from the source
	cluster.EnvironmentGitOwner = mergeString(cluster.EnvironmentGitOwner, to.Cluster.EnvironmentGitOwner)
	cluster.ExternalDNSSAName = mergeString(cluster.ExternalDNSSAName, to.Cluster.ExternalDNSSAName)
	cluster.GitKind = mergeString(cluster.GitKind, to.Cluster.GitKind)
	cluster.GitName = mergeString(cluster.GitName, to.Cluster.GitName)
	cluster.GitServer = mergeString(cluster.GitServer, to.Cluster.GitServer)
	cluster.Namespace = mergeString(cluster.Namespace, to.Cluster.Namespace)
	cluster.Provider = mergeString(cluster.Provider, to.Cluster.Provider)
	cluster.Registry = mergeString(cluster.Registry, to.Cluster.Registry)
	to.Cluster = cluster

	to.Vault = changes.Vault
	to.Storage = changes.Storage

	if changes.Ingress.TLS.Enabled {
		to.Ingress.TLS.Enabled = true
	}
	if changes.Ingress.TLS.Production {
		to.Ingress.TLS.Production = true
	}

	if cluster.ClusterName != "" {
		to.Cluster.ClusterName = cluster.ClusterName
	}
	if cluster.ProjectID != "" {
		to.Cluster.ProjectID = cluster.ProjectID
	}
	if cluster.Provider != "" {
		to.Cluster.Provider = cluster.Provider
	}
	if cluster.Region != "" {
		to.Cluster.Region = cluster.Region
	}
	if cluster.Zone != "" {
		to.Cluster.Zone = cluster.Zone
	}

	return nil
}

func mergeString(value1 string, value2 string) string {
	if value1 != "" {
		return value1
	}
	return value2
}
