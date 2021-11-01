package add

import (
	"fmt"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Adds a chart to the local 'helmfile.yaml' file
`)

	cmdExample = templates.Examples(`
		# adds a chart using the currently known repositories in the version stream or helmfile.yaml
		%s helmfile add --chart somerepo/mychart

		# adds a chart using a new repository URL with a custom version and namespace
		%s helmfile add --chart somerepo/mychart --repository https://acme.com/myrepo --namespace foo --version 1.2.3
	`)
)

// Options the options for the command
type Options struct {
	versionstreamer.Options
	helmfiles.ChartDetails

	GitCommitMessage string
	Helmfile         string
	DoGitCommit      bool
	Gitter           gitclient.Interface
}

// NewCmdHelmfileAdd creates a command object for the command
func NewCmdHelmfileAdd() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Adds a chart to the local 'helmfile.yaml' file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Options.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.GitCommitMessage, "commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")

	// chart flags
	cmd.Flags().StringVarP(&o.Chart, "chart", "c", "", "the name of the helm chart to add")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "jx", "the namespace to install the chart")
	cmd.Flags().StringVarP(&o.ReleaseName, "name", "", "", "the name of the helm release")
	cmd.Flags().StringVarP(&o.Repository, "repository", "r", "", "the helm chart repository URL of the chart")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "the version of the helm chart. If not specified the versionStream will be checked otherwise the latest version is used")
	cmd.Flags().StringArrayVarP(&o.Values, "values", "", nil, "the values files to add to the chart")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, "git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	if o.Chart == "" {
		return options.MissingOption("chart")
	}

	if o.Helmfile == "" {
		o.Helmfile = "helmfile.yaml"
	}

	o.Prefixes, err = o.Options.Resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", o.VersionStreamDir)
	}

	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: resolved charts and values from the version stream"
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	resolver := o.Options.Resolver
	if resolver == nil {
		return errors.Errorf("failed to create the VersionResolver")
	}

	hfNames, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to gather target helmfiles from %s", o.Dir)
	}

	editor, err := helmfiles.NewEditor(o.Dir, hfNames)
	if err != nil {
		return errors.Wrapf(err, "failed to create helmfile editor")
	}

	err = editor.AddChart(&o.ChartDetails)
	if err != nil {
		return errors.Wrapf(err, "failed to add chart")
	}

	err = editor.Save()
	if err != nil {
		return errors.Wrapf(err, "failed to save modified files")
	}

	_, err = o.Git().Command(o.Dir, "add", "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add helmfile changes to git in dir %s", o.Dir)
	}

	if !o.DoGitCommit {
		return nil
	}
	log.Logger().Infof("committing changes: %s", o.GitCommitMessage)
	err = o.GitCommit(o.Dir, o.GitCommitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes")
	}
	return nil
}

// Git returns the gitter - lazily creating one if required
func (o *Options) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.Gitter
}

func (o *Options) GitCommit(outDir, commitMessage string) error {
	gitter := o.Git()
	err := gitclient.CommitIfChanges(gitter, outDir, commitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes to git in dir %s", outDir)
	}
	return nil
}
