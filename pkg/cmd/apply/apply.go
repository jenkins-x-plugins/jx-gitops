package apply

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/filters"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Performs a gitops regeneration and apply on a cluster git repository

		If the last commit was a merge from a pull request the regeneration is skipped.

		Also the process detects if an ingress has changed (or similar changes) and retriggers another regeneration which typically is only required when installing for the first time or if no explicit domain name is being used and the LoadBalancer service has been removed.
`)

	cmdExample = templates.Examples(`
		# performs a regeneration and apply
		%s apply
	`)

	pathSeparator = string(os.PathSeparator)
)

// KptOptions the options for the command
type Options struct {
	Dir                     string
	Args                    []string
	LastCommitMessageFilter filters.StringFilter
	BatchMode               bool
	GitClient               gitclient.Interface
	CommandRunner           cmdrunner.CommandRunner
	GitCommandRunner        cmdrunner.CommandRunner
	Out                     io.Writer
	Err                     io.Writer
}

// NewCmdApply creates a command object for the command
func NewCmdApply() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "apply",
		Short:   "Performs a GitOps regeneration and apply on a cluster git repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to the git and make commands")

	o.LastCommitMessageFilter.AddFlags(cmd, "last-commit-msg", "last commit message")
	return cmd, o
}

// Validate validates the setup
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	lastCommitMessage, err := gitclient.GetLatestCommitMessage(o.GitClient, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to get last commit message")
	}
	lastCommitMessage = strings.TrimSpace(lastCommitMessage)
	log.Logger().Infof("found last commit message: %s", termcolor.ColorStatus(lastCommitMessage))

	regen := true
	if strings.Contains(lastCommitMessage, "/pipeline cancel") {
		log.Logger().Infof("last commit disabled regeneration so terminating")
		return nil
	}

	if strings.HasPrefix(lastCommitMessage, "Merge pull request") {
		log.Logger().Infof("last commit was a merge pull request so not regenerating")
		regen = false
	}

	if regen {
		_, err := o.Regenerate()
		if err != nil {
			return errors.Wrapf(err, "failed to regenerate")
		}

		c := &cmdrunner.Command{
			Dir:  o.Dir,
			Name: "make",
			Args: []string{"regen-phase-3"},
		}
		err = o.RunCommand(c)
		if err != nil {
			return errors.Wrapf(err, "failed to regenerate phase 3")
		}
	}
	return nil
}

// Regenerate regenerates the kubernetes resources
func (o *Options) Regenerate() (bool, error) {
	firstSha, err := gitclient.GetLatestCommitSha(o.GitClient, o.Dir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get the last commit sha")
	}

	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "make",
		Args: []string{"regen-phase-1"},
	}
	err = o.RunCommand(c)
	if err != nil {
		return false, errors.Wrapf(err, "failed to regenerate phase 1")
	}

	secondSha, err := gitclient.GetLatestCommitSha(o.GitClient, o.Dir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get the last commit sha")
	}

	if secondSha == firstSha {
		log.Logger().Infof("no commits so ")
		return false, nil
	}

	c = &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "make",
		Args: []string{"regen-phase-2"},
	}
	err = o.RunCommand(c)
	if err != nil {
		return false, errors.Wrapf(err, "failed to regenerate phase 2")
	}
	return true, nil
}

// Run runs the command
func (o *Options) RunCommand(c *cmdrunner.Command) error {
	log.Logger().Info(info(c.CLI()))
	c.Out = os.Stdout
	c.Err = os.Stderr
	_, err := o.CommandRunner(c)
	return err
}
