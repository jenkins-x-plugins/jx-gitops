package helmfile

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/add"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/report"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/resolve"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/status"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/structure"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/validate"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdHelmfile creates the new command
func NewCmdHelmfile() *cobra.Command {
	command := &cobra.Command{
		Use:     "helmfile",
		Short:   "Commands for working with helmfile",
		Aliases: []string{"helmfiles"},
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(add.NewCmdHelmfileAdd()))
	command.AddCommand(cobras.SplitCommand(move.NewCmdHelmfileMove()))
	command.AddCommand(cobras.SplitCommand(report.NewCmdHelmfileReport()))
	command.AddCommand(cobras.SplitCommand(resolve.NewCmdHelmfileResolve()))
	command.AddCommand(cobras.SplitCommand(status.NewCmdHelmfileStatus()))
	command.AddCommand(cobras.SplitCommand(structure.NewCmdHelmfileStructure()))
	command.AddCommand(cobras.SplitCommand(validate.NewCmdHelmfileValidate()))
	return command
}
