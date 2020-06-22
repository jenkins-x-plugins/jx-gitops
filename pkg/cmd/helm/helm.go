package helm

import (
	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdHelm creates the new command
func NewCmdHelm() *cobra.Command {
	command := &cobra.Command{
		Use:   "helm",
		Short: "Commands for working with helm charts",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(common.SplitCommand(NewCmdHelmTemplate()))
	command.AddCommand(common.SplitCommand(NewCmdHelmStream()))
	return command
}
