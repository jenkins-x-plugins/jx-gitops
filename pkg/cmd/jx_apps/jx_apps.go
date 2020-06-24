package jx_apps

import (
	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/v2/pkg/log"
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
	command.AddCommand(common.SplitCommand(NewCmdJxAppsTemplate()))
	return command
}
