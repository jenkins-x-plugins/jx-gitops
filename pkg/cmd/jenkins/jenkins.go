package jenkins

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/jenkins/add"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/jenkins/jobs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdJenkins creates the new command
func NewCmdJenkins() *cobra.Command {
	command := &cobra.Command{
		Use:   "jenkins",
		Short: "Commands for working with Jenkins GitOps configuration",
		Run: func(command *cobra.Command, _ []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Error(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(add.NewCmdJenkinsAdd()))
	command.AddCommand(cobras.SplitCommand(jobs.NewCmdJenkinsJobs()))
	return command
}
