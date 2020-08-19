package helmfile

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/add"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/resolve"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdHelmfile creates the new command
func NewCmdHelmfile() *cobra.Command {
	command := &cobra.Command{
		Use:     "helmfile",
		Short:   "Commands for working with helmfile",
		Aliases: []string{"helmfiles"},
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(add.NewCmdHelmfileAdd()))
	command.AddCommand(cobras.SplitCommand(move.NewCmdHelmfileMove()))
	command.AddCommand(cobras.SplitCommand(resolve.NewCmdHelmfileResolve()))
	return command
}
