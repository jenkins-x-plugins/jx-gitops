package push

import (
	"fmt"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
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

// Options the options for the command
type Options struct {
	scmhelpers.PullRequestOptions
	UserName          string
	UserEmail         string
	PullRequestBranch string

	BatchMode      bool
	DisableGitInit bool
	gitClient      gitclient.Interface
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
	cmd.Flags().BoolVarP(&o.IgnoreMissingPullRequest, "ignore-no-pr", "", false, "if an error is returned finding the Pull Request (maybe due to missing environment variables to find the PULL_NUMBER) just push to the current branch instead")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	o.BatchMode = true
	err := o.PullRequestOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	if kube.IsInCluster() && !o.DisableGitInit {
		err := o.InitGitConfigAndUser()
		if err != nil {
			return errors.Wrapf(err, "failed to init git")
		}
	}

	if o.PullRequestBranch == "" {
		pr, err := o.DiscoverPullRequest()
		if err != nil {
			if !o.IgnoreMissingPullRequest {
				return errors.Wrapf(err, "failed to discover pull request")
			}

			log.Logger().Infof("could not find Pull Request so assuming in a release pipeline. got: %s", err.Error())

			return o.pushToCurrentBranch()
		}
		o.PullRequestBranch = pr.Source
	}
	return o.pushToPullRequestBranch(o.PullRequestBranch)
}

func (o *Options) pushToPullRequestBranch(branch string) error {
	argSlices := [][]string{
		{
			"checkout", "-b", branch,
		},
		{
			"push", "origin", branch,
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

func (o *Options) pushToCurrentBranch() error {
	branch, err := gitclient.Branch(o.GitClient(), o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to determine git branch in dir %s", o.Dir)
	}
	if branch == "" {
		branch = "master"
	}

	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "git",
		Args: []string{"push", "origin", branch},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run command %s", c.CLI())
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
