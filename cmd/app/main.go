// +build !windows

package app

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd"
)

// Run runs the command, if args are not nil they will be set on the command
func Run(args []string) error {
	command := cmd.Main()
	if args != nil {
		args = args[1:]
		command.SetArgs(args)
	}
	return command.Execute()
}
