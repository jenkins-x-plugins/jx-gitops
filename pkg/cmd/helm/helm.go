package helm

import (
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
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
	command.AddCommand(cobras.SplitCommand(NewCmdHelmTemplate()))
	command.AddCommand(cobras.SplitCommand(NewCmdHelmStream()))
	return command
}
