package push_test

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/pr/push"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestPush(t *testing.T) {
	_, pp := push.NewCmdPullRequestPush()

	prNumber := 123
	repo := "myorg/myrepo"
	prBranch := "my-pr-branch-name"

	runner := &fakerunner.FakeRunner{}
	pp.CommandRunner = runner.Run
	pp.Number = prNumber
	pp.Repository = repo

	scmClient, fakeData := fake.NewDefault()
	pp.ScmClient = scmClient
	fakeData.PullRequests[prNumber] = &scm.PullRequest{
		Number: prNumber,
		Title:  "my awesome pull request",
		Body:   "some text",
		Source: prBranch,
	}

	err := pp.Run()
	require.NoError(t, err, "failed to run pull request push")

	assert.Equal(t, prBranch, pp.Branch, "pr.Branch name")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git pull",
		},
		fakerunner.FakeResult{
			CLI: "git checkout -b " + prBranch,
		},
		fakerunner.FakeResult{
			CLI: "git push origin " + prBranch,
		},
	)
}
