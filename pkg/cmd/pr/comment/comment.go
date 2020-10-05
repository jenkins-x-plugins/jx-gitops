package comment

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Adds a comment to the current pull request
`)

	cmdExample = templates.Examples(`
		# add comment
		%s pr comment "Message from Jenkins"
	`)
)

// Options the options for the command
type Options struct {
	scmhelpers.PullRequestOptions

	Comment string
	Result  *scm.PullRequest
}

// NewCmdPullRequestComment creates a command object for the command
func NewCmdPullRequestComment() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "comment",
		Short:   "Add comment to the pull request",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {

			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.PullRequestOptions.AddFlags(cmd)

	cmd.Flags().StringVar(&o.Comment, "comment", "", "comment to add")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.PullRequestOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}
	pr, err := o.DiscoverPullRequest()
	if err != nil {
		return errors.Wrapf(err, "failed to discover the pull request")
	}
	if pr == nil {
		return errors.Errorf("no Pull Request could be found for %d in repository %s", o.Number, o.Repository)
	}
	return o.commentPullRequest(pr)
}

func (o *Options) commentPullRequest(pr *scm.PullRequest) error {
	o.Result = pr

	ctx := context.Background()
	comment := &scm.CommentInput{Body: o.Comment}
	_, _, err := o.ScmClient.PullRequests.CreateComment(ctx, o.FullRepositoryName, o.Number, comment)
	prName := "#" + strconv.Itoa(o.Number)
	if err != nil {
		return errors.Wrapf(err, "failed to comment on pull request %s on repository %s", prName, o.FullRepositoryName)
	}
	log.Logger().Infof("commented on pull request %s on repository %s", prName, o.FullRepositoryName)

	return nil

}
