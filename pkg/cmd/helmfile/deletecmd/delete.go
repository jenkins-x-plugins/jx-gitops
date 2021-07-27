package deletecmd

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Options the flags for updating webhooks
type Options struct {
	options.BaseOptions

	Details          helmfiles.ChartDetails
	Dir              string
	Helmfile         string
	GitCommitMessage string
	DoGitCommit      bool

	Gitter        gitclient.Interface
	CommandRunner cmdrunner.CommandRunner
}

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Deletes a chart from the helmfiles in one or all namespaces

`)

	cmdExample = templates.Examples(`
		# deletes the chart from all namespaces
		jx gitops helmfile delete --chart my-chart

		# deletes the chart from a specific namespace
		jx gitops helmfile delete --chart my-chart --namespace jx-staging

		# deletes the chart with the repo prefix from a specific namespace
		jx gitops helmfile delete --chart myrepo/my-chart --namespace jx-staging
`)
)

func NewCmdHelmfileDelete() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"remove", "rm", "del"},
		Short:   "Deletes a chart from the helmfiles in one or all namespaces",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	o.BaseOptions.AddBaseFlags(cmd)

	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.GitCommitMessage, "commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the helmfile.yaml and helmfiles directory")

	// chart flags
	cmd.Flags().StringVarP(&o.Details.Chart, "chart", "c", "", "the name of the helm chart to remove")
	cmd.Flags().StringVarP(&o.Details.Namespace, "namespace", "n", "", "the namespace to remove the chart from. If blank then remove from all namespaces")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, "git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	if o.Details.Chart == "" {
		return options.MissingOption("chart")
	}

	if o.Helmfile == "" {
		o.Helmfile = "helmfile.yaml"
	}

	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: remove chart " + o.Details.Chart
	}
	return nil
}

// Run runs the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	hfNames, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to gather target helmfiles from %s", o.Dir)
	}

	editor, err := helmfiles.NewEditor(o.Dir, hfNames)
	if err != nil {
		return errors.Wrapf(err, "failed to create helmfile editor")
	}

	err = editor.DeleteChart(&o.Details)
	if err != nil {
		return errors.Wrapf(err, "failed to add chart %s", o.Details.Chart)
	}

	err = editor.Save()
	if err != nil {
		return errors.Wrapf(err, "failed to save modified files")
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

func (o *Options) GitCommit(outDir string, commitMessage string) error {
	gitter := o.Git()
	err := gitclient.CommitIfChanges(gitter, outDir, commitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes to git in dir %s", outDir)
	}
	return nil
}
