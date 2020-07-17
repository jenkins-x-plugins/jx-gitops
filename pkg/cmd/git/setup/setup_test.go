package setup_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/setup"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
)

func TestGitSetup(t *testing.T) {
	_, o := setup.NewCmdGitSetup()

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run
	o.UserEmail = "fakeuser@googlegroups.com"
	o.UserName = "fakeusername"

	err := o.Run()
	require.NoError(t, err, "failed to run git setup")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git config --get user.name",
		},
		fakerunner.FakeResult{
			CLI: "git config --get user.email",
		},
		fakerunner.FakeResult{
			CLI: "git config --global --add user.name fakeusername",
		},
		fakerunner.FakeResult{
			CLI: "git config --global --add user.email fakeuser@googlegroups.com",
		},
		fakerunner.FakeResult{
			CLI: "git config --global credential.helper store",
		},
	)
}
