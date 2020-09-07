package sa

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/edit"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/merge"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/resolve"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdServiceAccount creates the new command
func NewCmdServiceAccount() *cobra.Command {
	command := &cobra.Command{
		Use:     "sa",
		Short:   "Commands for working with kubernetes ServiceAccount resources",
		Aliases: []string{"serviceaccount", "serviceaccounts"},
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(resolve.NewCmdRequirementsResolve()))
	command.AddCommand(cobras.SplitCommand(edit.NewCmdRequirementsEdit()))
	command.AddCommand(cobras.SplitCommand(merge.NewCmdRequirementsMerge()))
	return command
}
