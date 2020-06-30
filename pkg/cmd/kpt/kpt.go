package kpt

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/recreate"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdHelm creates the new command
func NewCmdKpt() *cobra.Command {
	command := &cobra.Command{
		Use:   "kpt",
		Short: "Commands for working with kpt packages",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(recreate.NewCmdKptRecreate()))
	command.AddCommand(cobras.SplitCommand(update.NewCmdKptUpdate()))
	return command
}
