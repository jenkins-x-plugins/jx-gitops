package requirement

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/edit"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/merge"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/publish"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/resolve"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

var requirementRetriableErrors = []string{
	"dial tcp \\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}:\\d+: i/o timeout",
}

// NewCmdRequirement creates the new command
func NewCmdRequirement() *cobra.Command {
	command := &cobra.Command{
		Use:     "requirement",
		Short:   "Commands for working with jx-requirements.yml",
		Aliases: []string{"req", "requirements"},
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(edit.NewCmdRequirementsEdit()))
	command.AddCommand(helper.RetryOnErrorCommand(cobras.SplitCommand(merge.NewCmdRequirementsMerge()), helper.RegexRetryFunction(requirementRetriableErrors)))
	command.AddCommand(helper.RetryOnErrorCommand(cobras.SplitCommand(resolve.NewCmdRequirementsResolve()), helper.RegexRetryFunction(requirementRetriableErrors)))
	command.AddCommand(cobras.SplitCommand(publish.NewCmdRequirementsPublish()))
	return command
}
