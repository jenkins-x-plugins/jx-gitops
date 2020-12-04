package fakerunners

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
)

// NewFakeRunnerWithGitClone creates a fake runner which can still git clone
func NewFakeRunnerWithGitClone() *fakerunner.FakeRunner {
	return &fakerunner.FakeRunner{
		CommandRunner: func(command *cmdrunner.Command) (string, error) {
			if command.Name == "git" && len(command.Args) > 1 && command.Args[0] == "clone" {
				return cmdrunner.DefaultCommandRunner(command)
			}
			return "fake " + command.CLI(), nil
		},
	}
}
