package fakerunners

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
)

// NewFakeRunnerWithGitClone creates a fake runner which can still git clone
func NewFakeRunnerWithGitClone() *fakerunner.FakeRunner {
	return &fakerunner.FakeRunner{
		CommandRunner: func(command *cmdrunner.Command) (string, error) {
			if command.Name == "git" && len(command.Args) > 1 {
				if command.Args[0] == "clone" || command.Args[0] == "sparse-checkout" {
					return cmdrunner.DefaultCommandRunner(command)
				}
			} else if command.Args[0] == "checkout" {
				return cmdrunner.DefaultCommandRunner(command)
			}
			return "fake " + command.CLI(), nil
		},
	}
}
