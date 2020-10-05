package add

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Add one or more repositories to the SourceConfig
`)

	cmdExample = templates.Examples(`
		# creates any missing SourceConfig resources  
		%s repository add https://github.com/myorg/myrepo.git
	`)
)

// LabelOptions the options for the command
type Options struct {
	kyamls.Filter
	Args         []string
	Dir          string
	ConfigFile   string
	Scheduler    string
	Namespace    string
	JXClient     versioned.Interface
	ExplicitMode bool
}

// NewCmdAddRepository creates a command object for the command
func NewCmdAddRepository() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Add one or more git URLs to the source configuration",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory look for the 'jx-requirements.yml` file")
	cmd.Flags().StringVarP(&o.ConfigFile, "config", "c", "", "the configuration file to load for the repository configurations. If not specified we look in .jx/gitops/source-repositories.yaml")
	cmd.Flags().StringVarP(&o.Scheduler, "scheduler", "s", "", "the name of the Scheduler to use for the repository")
	cmd.Flags().BoolVarP(&o.ExplicitMode, "explicit", "e", false, "Explicit mode: always populate all the fields even if they can be deduced. e.g. the git URLs for each repository are not absolutely necessary and are omitted by default are populated if this flag is enabled")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace to discover SourceRepository resources to default the GitKind. If not specified then use the current namespace")

	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run transforms the YAML files
func (o *Options) Run() error {
	if o.ConfigFile == "" {
		o.ConfigFile = filepath.Join(o.Dir, ".jx", "gitops", v1alpha1.SourceConfigFileName)
	}
	if len(o.Args) == 0 {
		return errors.Errorf("missing git URL argument")
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

	for _, arg := range o.Args {
		err = o.ensureSourceRepositoryExists(config, arg)
		if err != nil {
			return errors.Wrapf(err, "failed to ")
		}
	}

	sourceconfigs.SortConfig(config)
	sourceconfigs.EnrichConfig(config)

	if !o.ExplicitMode {
		sourceconfigs.DryConfig(config)
	}
	dir := filepath.Dir(o.ConfigFile)
	err = os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}

	err = yamls.SaveFile(config, o.ConfigFile)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.ConfigFile)
	}
	action := "created"
	if exists {
		action = "modified"
	}
	log.Logger().Infof("%s file %s", action, termcolor.ColorInfo(o.ConfigFile))

	return nil
}

func (o *Options) ensureSourceRepositoryExists(config *v1alpha1.SourceConfig, gitURL string) error {
	if gitURL == "" {
		return errors.Errorf("empty git URL")
	}
	gitInfo, err := giturl.ParseGitURL(gitURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse git URL %s", gitURL)
	}

	gitServerURL := gitInfo.HostURL()
	gitKind, err := scmhelpers.DiscoverGitKind(o.JXClient, o.Namespace, gitServerURL)
	if err != nil {
		return errors.Wrapf(err, "failed to discover the git kind")
	}

	group := sourceconfigs.GetOrCreateGroup(config, gitKind, gitServerURL, gitInfo.Organisation)
	repo := sourceconfigs.GetOrCreateRepository(group, gitInfo.Name)

	if o.Scheduler != "" && o.Scheduler != group.Scheduler {
		repo.Scheduler = o.Scheduler
	}
	return nil
}
