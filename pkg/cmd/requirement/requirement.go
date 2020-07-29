package requirement

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/edit"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/resolve"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdRequirement creates the new command
func NewCmdRequirement() *cobra.Command {
	command := &cobra.Command{
		Use:     "requirement",
		Short:   "Commands for working with jx-requirements.yml",
		Aliases: []string{"req", "requirements"},
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(resolve.NewCmdRequirementsResolve()))
	command.AddCommand(cobras.SplitCommand(edit.NewCmdRequirementsEdit()))
	return command
}
