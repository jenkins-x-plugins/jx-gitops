package setup

import (
	"fmt"

	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"k8s.io/client-go/rest"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Sets up git to ensure the git user name and email is setup.

This is typically used in a pipeline to ensure git can do commits.
`)

	cmdExample = templates.Examples(`
		%s git setup 
	`)
)

// Options the options for the command
type Options struct {
	Dir           string
	UserName      string
	UserEmail     string
	CommandRunner cmdrunner.CommandRunner
	gitClient     gitclient.Interface
}

// NewCmdGitSetup creates a command object for the command
func NewCmdGitSetup() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "setup",
		Short:   "Sets up git to ensure the git user name and email is setup",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", "", "the directory to run the git push command from")
	cmd.Flags().StringVarP(&o.UserName, "name", "n", "", "the git user name to use if one is not setup")
	cmd.Flags().StringVarP(&o.UserEmail, "email", "e", "", "the git user email to use if one is not setup")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	gitClient := o.GitClient()
	_, _, err := gitclient.EnsureUserAndEmailSetup(gitClient, o.Dir, o.UserName, o.UserEmail)
	if err != nil {
		return errors.Wrapf(err, "failed to setup git user and email")
	}
	err = gitclient.SetCredentialHelper(gitClient, "")
	if err != nil {
		return errors.Wrapf(err, "failed to setup credential store")
	}
	return nil
}

func (o *Options) GitClient() gitclient.Interface {
	if o.gitClient == nil {
		o.gitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.gitClient
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
