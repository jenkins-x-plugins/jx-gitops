package apps

import (
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdApps creates the new command
func NewCmdApps() *cobra.Command {
	command := &cobra.Command{
		Use:     "apps",
		Short:   "Commands for working with jx-apps.yml",
		Aliases: []string{"jx-apps", "app"},
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
