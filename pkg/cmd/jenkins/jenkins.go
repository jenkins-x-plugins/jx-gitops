package jenkins

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm/release"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdJenkins creates the new command
func NewCmdJenkins() *cobra.Command {
	command := &cobra.Command{
		Use:   "jenkins",
		Short: "Commands for working with Jenkins GitOps configuration",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(release.NewCmdHelmRelease()))
	return command
}
