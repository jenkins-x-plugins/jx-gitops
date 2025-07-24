package fakerunners

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
)

// NewFakeRunnerWithGitClone creates a fake runner which can still git clone, sparse-checkout and checkout
func NewFakeRunnerWithGitClone() *fakerunner.FakeRunner {
	return &fakerunner.FakeRunner{
		CommandRunner: func(command *cmdrunner.Command) (string, error) {
			if command.Name == "git" && len(command.Args) > 0 &&
				(command.Args[0] == "clone" || command.Args[0] == "checkout" || command.Args[0] == "sparse-checkout") {
				return cmdrunner.DefaultCommandRunner(command)
			}
			return "fake " + command.CLI(), nil
		},
	}
}
