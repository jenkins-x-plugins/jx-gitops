package condition_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/condition"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
)

func TestConditionMatches(t *testing.T) {
	commitMessage := "Merge pull request #123 blah blah"

	_, o := condition.NewCmdCondition()
	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "git" {
				return commitMessage, nil
			}
			return "", nil
		},
	}
	o.CommandRunner = runner.Run
	o.LastCommitMessageFilter.Prefix = "Merge pull request"
	o.Args = []string{"make all"}
	err := o.Run()
	require.NoError(t, err, "failed to run conditional")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git log -1 --pretty=%B",
		},
		fakerunner.FakeResult{
			CLI: "make all",
		},
	)
}

func TestConditionNotMatch(t *testing.T) {
	commitMessage := "something random"

	_, o := condition.NewCmdCondition()
	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "git" {
				return commitMessage, nil
			}
			return "", nil
		},
	}
	o.CommandRunner = runner.Run
	o.LastCommitMessageFilter.Prefix = "Merge pull request"
	o.Args = []string{"make all"}
	err := o.Run()
	require.NoError(t, err, "failed to run conditional")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git log -1 --pretty=%B",
		},
	)
}
