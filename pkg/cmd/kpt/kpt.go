package kpt

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kpt/recreate"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdHelm creates the new command
func NewCmdKpt() *cobra.Command {
	command := &cobra.Command{
		Use:   "kpt",
		Short: "Commands for working with kpt packages",
		Run: func(command *cobra.Command, _ []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Error(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(recreate.NewCmdKptRecreate()))
	command.AddCommand(cobras.SplitCommand(update.NewCmdKptUpdate()))
	return command
}
