package push

import (
	"fmt"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/kube"
	"github.com/jenkins-x/jx-helpers/pkg/scmhelpers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Pushes the current git directory to the branch used to create the Pull Request
`)

	cmdExample = templates.Examples(`
		# pushes the current directories git contents to the branch used to create the current PR via $BRANCH_NAME
		%s pr push 
	`)
)

// KptOptions the options for the command
type Options struct {
	scmhelpers.PullRequestOptions
	UserName  string
	UserEmail string
	BatchMode bool
	gitClient gitclient.Interface
}

// NewCmdPullRequestPush creates a command object for the command
func NewCmdPullRequestPush() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "push",
		Short:   "Pushes the current git directory to the branch used to create the Pull Request",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	o.PullRequestOptions.AddFlags(cmd)
	cmd.Flags().StringVarP(&o.UserName, "name", "", "", "the git user name to use if one is not setup")
	cmd.Flags().StringVarP(&o.UserEmail, "email", "", "", "the git user email to use if one is not setup")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	o.BatchMode = true
	err := o.PullRequestOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}
	if kube.IsInCluster() {
		err := o.InitGitConfigAndUser()
		if err != nil {
			return errors.Wrapf(err, "failed to init git")
		}
	}
	return o.pushToBranch()
}

func (o *Options) pushToBranch() error {
	argSlices := [][]string{
		{
			"checkout", "-b", o.Branch,
		},
		{
			"push", "origin", o.Branch,
		},
	}

	for _, args := range argSlices {
		c := &cmdrunner.Command{
			Dir:  o.Dir,
			Name: "git",
			Args: args,
		}
		_, err := o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run command %s", c.CLI())
		}
	}
	return nil
}

func (o *Options) GitClient() gitclient.Interface {
	if o.gitClient == nil {
		o.gitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.gitClient
}

func (o *Options) InitGitConfigAndUser() error {
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
