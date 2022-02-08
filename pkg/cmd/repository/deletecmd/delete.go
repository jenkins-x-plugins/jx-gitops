package deletecmd

import (
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Add one or more repositories to the SourceConfig
`)

	cmdExample = templates.Examples(`
		# deletes a repository by name from the '.jx/gitops/source-config.yaml' file
		jx gitops repository delete --name myrepo

		# deletes a repository by name and owner from the '.jx/gitops/source-config.yaml' file
		jx gitops repository delete --name myrepo --owner myowner
	`)
)

// LabelOptions the options for the command
type Options struct {
	Owner        string
	Name         string
	Dir          string
	ConfigFile   string
	Namespace    string
	JXClient     versioned.Interface
	ExplicitMode bool
}

// NewCmdDeleteRepository creates a command object for the command
func NewCmdDeleteRepository() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"remove", "rm", "del"},
		Short:   "Deletes a repository from the source configuration",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory look for the 'jx-requirements.yml` file")
	cmd.Flags().StringVarP(&o.ConfigFile, "config", "c", "", "the configuration file to load for the repository configurations. If not specified we look in .jx/gitops/source-repositories.yaml")
	cmd.Flags().StringVarP(&o.Name, "name", "n", "", "the name of the repository to remove")
	cmd.Flags().StringVarP(&o.Owner, "owner", "o", "", "the owner of the repository to remove")
	return cmd, o
}

// Run transforms the YAML files
func (o *Options) Run() error {
	if o.ConfigFile == "" {
		o.ConfigFile = filepath.Join(o.Dir, ".jx", "gitops", v1alpha1.SourceConfigFileName)
	}
	if o.Name == "" {
		return options.MissingOption("name")
	}

	exists, err := files.FileExists(o.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", o.ConfigFile)
	}

	config := &v1alpha1.SourceConfig{}

	if !exists {
		log.Logger().Infof("file %s does not exist", termcolor.ColorStatus(o.ConfigFile))
		return nil
	}

	err = yamls.LoadFile(o.ConfigFile, config)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", o.ConfigFile)
	}

	modified := sourceconfigs.RemoveRepository(config, o.Owner, o.Name)
	if !modified {
		log.Logger().Infof("repository %s not found in file %s", info(o.Name), info(o.ConfigFile))
		return nil
	}

	err = yamls.SaveFile(config, o.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.ConfigFile)
	}

	log.Logger().Infof("modified file %s", info(o.ConfigFile))
	return nil
}
