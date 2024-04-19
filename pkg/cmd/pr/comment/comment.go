package comment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	// Create a new comment even if it already exists
	CreateCommentStrategy = "create"
	// Create a new comment only if it doesn't already exist
	CreateIfNotExistsCommentStrategy = "create-if-not-exists"
	// Delete a comment if it exists and create it again
	DeleteAndCreateCommentStrategy = "delete-and-create"
)

var (
	cmdLong = templates.LongDesc(`
		Adds a comment to the current pull request
`)

	cmdExample = templates.Examples(`
		# add comment
		%s pr comment -c "message from Jenkins X pipeline"
	`)

	availableStrategies = []string{
		CreateCommentStrategy,
		CreateIfNotExistsCommentStrategy,
		DeleteAndCreateCommentStrategy,
	}
)

// Options the options for the command
type Options struct {
	scmhelpers.PullRequestOptions

	Comment  string
	Result   *scm.PullRequest
	Strategy string
}

// NewCmdPullRequestComment creates a command object for the command
func NewCmdPullRequestComment() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "comment",
		Short:   "Add comment to the pull request",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.PullRequestOptions.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.Comment, "comment", "c", "", "comment to add")
	cmd.Flags().StringVarP(&o.Strategy, "strategy", "s", CreateCommentStrategy, fmt.Sprintf("comment strategy to choose (%s)", strings.Join(availableStrategies, ", ")))
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
	return o.commentPullRequestWithStrategy(context.Background())
}

func (o *Options) commentPullRequestWithStrategy(ctx context.Context) error {
	switch o.Strategy {
	case CreateCommentStrategy:
		return o.create(ctx)

	case CreateIfNotExistsCommentStrategy:
		return o.createIfNotExists(ctx)

	case DeleteAndCreateCommentStrategy:
		return o.deleteAndCreate(ctx)

	default:
		return o.create(ctx)
	}
}

func (o *Options) create(ctx context.Context) error {
	prName := "#" + strconv.Itoa(o.Number)
	comment := &scm.CommentInput{Body: o.Comment}
	_, _, err := o.ScmClient.PullRequests.CreateComment(ctx, o.FullRepositoryName, o.Number, comment)
	if err != nil {
		return errors.Wrapf(err, "failed to comment on pull request %s on repository %s", prName, o.FullRepositoryName)
	}
	log.Logger().Infof("commented on pull request %s on repository %s", prName, o.FullRepositoryName)
	return nil
}

func (o *Options) list(ctx context.Context) ([]*scm.Comment, error) {
	comments, _, err := o.ScmClient.PullRequests.ListComments(ctx, o.FullRepositoryName, o.Number, &scm.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list comments on pull request #%d on repository %s", o.Number, o.FullRepositoryName)
	}

	return comments, nil
}

func (o *Options) delete(ctx context.Context, comments []*scm.Comment) error {
	prName := "#" + strconv.Itoa(o.Number)

	for _, comment := range comments {
		_, err := o.ScmClient.PullRequests.DeleteComment(ctx, o.FullRepositoryName, o.Number, comment.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to delete comment with ID %d on pull request %s on repository %s", comment.ID, prName, o.FullRepositoryName)
		}
		log.Logger().Infof("deleted comment with ID %d on pull request %s on repository %s", comment.ID, prName, o.FullRepositoryName)
	}

	return nil
}

func (o *Options) createIfNotExists(ctx context.Context) error {
	existingComments, err := o.list(ctx)
	if err != nil {
		return err
	}

	for i := range existingComments {
		comment := existingComments[i]
		if o.ScmClient.Username == comment.Author.Login && comment.Body == o.Comment {
			log.Logger().Infof("Similar comment already exists on pull request #%d on repository %s", o.Number, o.FullRepositoryName)
			return nil
		}
	}

	return o.create(ctx)
}

func (o *Options) deleteAndCreate(ctx context.Context) error {
	existingComments, err := o.list(ctx)
	if err != nil {
		return err
	}

	similarComments := make([]*scm.Comment, 0)
	for i := range existingComments {
		comment := existingComments[i]
		if o.ScmClient.Username == comment.Author.Login && comment.Body == o.Comment {
			similarComments = append(similarComments, comment)
		}
	}

	if len(similarComments) > 0 {
		if err := o.delete(ctx, similarComments); err != nil {
			return err
		}
	}

	return o.create(ctx)
}
