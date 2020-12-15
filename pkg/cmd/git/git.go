package git

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/clone"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/get"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/merge"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/setup"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdGit creates the new command
func NewCmdGit() *cobra.Command {
	command := &cobra.Command{
		Use:   "git",
		Short: "Commands for working with Git",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(clone.NewCmdGitClone()))
	command.AddCommand(cobras.SplitCommand(get.NewCmdGitGet()))
	command.AddCommand(cobras.SplitCommand(merge.NewCmdGitMerge()))
	command.AddCommand(cobras.SplitCommand(setup.NewCmdGitSetup()))
	return command
}
