package plugin

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/plugin/get"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/plugin/upgrade"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdPlugin creates the new command
func NewCmdPlugin() *cobra.Command {
	command := &cobra.Command{
		Use:   "plugin",
		Short: "Commands for working with plugins",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(get.NewCmdPluginGet()))
	command.AddCommand(cobras.SplitCommand(upgrade.NewCmdUpgradePlugins()))
	return command
}
