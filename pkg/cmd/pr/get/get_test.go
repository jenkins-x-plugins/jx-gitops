package get_test

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/pr/get"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestGet(t *testing.T) {
	_, pp := get.NewCmdPullRequestGet()

	prNumber := 123
	repo := "myorg/myrepo"
	prBranch := "my-pr-branch-name"
	expectedHeadClone := "https://github.com/jenkins-x-labs-bot/myrepo.git"

	runner := &fakerunner.FakeRunner{}
	pp.CommandRunner = runner.Run
	pp.SourceURL = "https://github.com/" + repo
	pp.Number = prNumber
	pp.Branch = prBranch

	scmClient, fakeData := fake.NewDefault()
	pp.ScmClient = scmClient
	fakeData.PullRequests[prNumber] = &scm.PullRequest{
		Number: prNumber,
		Title:  "my awesome pull request",
		Body:   "some text",
		Source: prBranch,
		Head: scm.PullRequestBranch{
			Repo: scm.Repository{
				Clone: expectedHeadClone,
			},
		},
	}

	err := pp.Run()
	require.NoError(t, err, "failed to run pull request push")

	require.NotNil(t, pp.Result, "no PullRequest found")
	assert.Equal(t, expectedHeadClone, pp.Result.Head.Repo.Clone, "pr.Head.Repo.Clone")
}
