package comment_test

import (
	"context"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/pr/comment"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestComment(t *testing.T) {
	_, o := comment.NewCmdPullRequestComment()

	prNumber := 123
	repo := "myorg/myrepo"
	prBranch := "my-pr-branch-name"
	expectedComment := "hello from a pipeline"

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run
	o.SourceURL = "https://github.com/" + repo
	o.Number = prNumber
	o.Branch = prBranch
	o.Comment = expectedComment

	scmClient, fakeData := fake.NewDefault()
	o.ScmClient = scmClient
	fakeData.PullRequests[prNumber] = &scm.PullRequest{
		Number: prNumber,
		Title:  "my awesome pull request",
		Body:   "some text",
		Source: prBranch,
	}

	err := o.Run()
	require.NoError(t, err, "failed to run ")

	ctx := context.Background()
	comments, _, err := o.ScmClient.PullRequests.ListComments(ctx, repo, prNumber, scm.ListOptions{})
	require.NoError(t, err, "failed to list comments")
	require.NotEmpty(t, comments, "should have some comments")

	lastComment := comments[len(comments)-1]
	assert.Equal(t, expectedComment, lastComment.Body, "lastComment.Body")
	t.Logf("pull request #%d, has comment: %s\n", prNumber, lastComment.Body)
}
