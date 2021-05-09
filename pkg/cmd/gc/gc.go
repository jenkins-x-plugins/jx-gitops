package gc

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/gc/activities"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/gc/pods"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdGC creates the new command
func NewCmdGC() *cobra.Command {
	command := &cobra.Command{
		Use:   "gc",
		Short: "Commands for garbage collecting resources",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(activities.NewCmdGCActivities()))
	command.AddCommand(cobras.SplitCommand(pods.NewCmdGCPods()))
	return command
}
