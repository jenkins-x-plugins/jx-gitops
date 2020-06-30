package jx_apps

import (
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdJxApps creates the new command
func NewCmdJxApps() *cobra.Command {
	command := &cobra.Command{
		Use:   "jx-apps",
		Short: "Commands for working with jx-apps.yml",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(NewCmdJxAppsTemplate()))
	return command
}
