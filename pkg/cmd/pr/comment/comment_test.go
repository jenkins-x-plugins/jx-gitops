package comment_test

import (
	"context"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/pr/comment"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestCommentStrategies(t *testing.T) {
	scenarios := []struct {
		name                 string
		author               string
		strategy             string
		newCommentText       string
		existingCommentsText []string
		expectedCommentCount int
	}{
		{
			name:                 "CreateCommentStrategy-Existing-DifferentUser",
			author:               "CreateCommentStrategy-Existing-DifferentUser",
			strategy:             comment.CreateCommentStrategy,
			newCommentText:       "comment for CreateCommentStrategy",
			existingCommentsText: []string{"comment for CreateCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "CreateIfNotExistsCommentStrategy-Existing-DifferentUser",
			author:               "CreateIfNotExistsCommentStrategy-Existing-DifferentUser",
			strategy:             comment.CreateIfNotExistsCommentStrategy,
			newCommentText:       "comment for CreateIfNotExistsCommentStrategy",
			existingCommentsText: []string{"comment for CreateIfNotExistsCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "DeleteAndCreateCommentStrategy-Existing-DifferentUser",
			author:               "DeleteAndCreateCommentStrategy-Existing-DifferentUser",
			strategy:             comment.DeleteAndCreateCommentStrategy,
			newCommentText:       "comment for DeleteAndCreateCommentStrategy",
			existingCommentsText: []string{"comment for DeleteAndCreateCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "CreateCommentStrategy-Existing-SimilarUser",
			author:               "",
			strategy:             comment.CreateCommentStrategy,
			newCommentText:       "comment for CreateCommentStrategy",
			existingCommentsText: []string{"comment for CreateCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "CreateCommentStrategy-NotExisting-SimilarUser",
			author:               "",
			strategy:             comment.CreateCommentStrategy,
			newCommentText:       "new-comment for CreateCommentStrategy",
			existingCommentsText: []string{"comment for CreateCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "CreateIfNotExistsCommentStrategy-Existing-SimilarUser",
			author:               "",
			strategy:             comment.CreateIfNotExistsCommentStrategy,
			newCommentText:       "existing-comment for CreateIfNotExistsCommentStrategy",
			existingCommentsText: []string{"existing-comment for CreateIfNotExistsCommentStrategy"},
			expectedCommentCount: 1,
		},
		{
			name:                 "CreateIfNotExistsCommentStrategy-NotExisting-SimilarUser",
			author:               "",
			strategy:             comment.CreateIfNotExistsCommentStrategy,
			newCommentText:       "new-comment for CreateIfNotExistsCommentStrategy",
			existingCommentsText: []string{"existing-comment for CreateIfNotExistsCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "DeleteAndCreateCommentStrategy-Existing-SimilarUser",
			author:               "",
			strategy:             comment.DeleteAndCreateCommentStrategy,
			newCommentText:       "comment for DeleteAndCreateCommentStrategy",
			existingCommentsText: []string{"comment for DeleteAndCreateCommentStrategy", "existing-comment for DeleteAndCreateCommentStrategy"},
			expectedCommentCount: 2,
		},
		{
			name:                 "DeleteAndCreateCommentStrategy-NotExisting-SimilarUser",
			author:               "",
			strategy:             comment.DeleteAndCreateCommentStrategy,
			newCommentText:       "comment for DeleteAndCreateCommentStrategy",
			existingCommentsText: []string{"existing-comment for DeleteAndCreateCommentStrategy"},
			expectedCommentCount: 2,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			o := setupOptionsAndData(scenario.strategy, scenario.newCommentText, scenario.existingCommentsText, scenario.author)
			testCommentStrategy(t, o, scenario.expectedCommentCount)
		})
	}
}

func setupOptionsAndData(strategy string, newCommentText string, existingCommentsText []string, author string) *comment.Options {
	_, o := comment.NewCmdPullRequestComment()

	prNumber := 123
	repo := "myorg/myrepo"
	prBranch := "my-pr-branch-name"

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run
	o.SourceURL = "https://github.com/" + repo
	o.Number = prNumber
	o.Branch = prBranch
	o.Comment = newCommentText
	o.Strategy = strategy

	scmClient, fakeData := fake.NewDefault()
	if author != "" {
		fakeData.CurrentUser.Name = author
		fakeData.CurrentUser.Login = author
	}
	o.ScmClient = scmClient

	fakeData.PullRequests[prNumber] = &scm.PullRequest{
		Number: prNumber,
		Title:  "my-pr",
		Body:   "body",
	}

	// Add existing comments to the pull request
	for _, existingCommentText := range existingCommentsText {
		fakeData.PullRequestComments[prNumber] = append(fakeData.PullRequestComments[prNumber], &scm.Comment{
			Author: scm.User{
				Login: fakeData.CurrentUser.Name,
				Name:  fakeData.CurrentUser.Name,
			},
			Body: existingCommentText,
		})
	}

	return o
}

func testCommentStrategy(t *testing.T, o *comment.Options, expectedCommentCount int) {
	err := o.Run()
	require.NoError(t, err, "failed to run ")

	ctx := context.Background()
	comments, _, err := o.ScmClient.PullRequests.ListComments(ctx, o.Repository, o.Number, scm.ListOptions{})
	require.NoError(t, err, "failed to list comments")
	require.NotEmpty(t, comments, "should have some comments")

	lastComment := comments[len(comments)-1]

	assert.Equal(t, expectedCommentCount, len(comments), "expectedCommentCount")
	assert.Equal(t, o.Comment, lastComment.Body, "lastComment.Body")

	t.Logf("pull request #%d, has comments: %s\n", o.Number, lastComment.Body)
}
