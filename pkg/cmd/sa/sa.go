package sa

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/sa/secret"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdServiceAccount creates the new command
func NewCmdServiceAccount() *cobra.Command {
	command := &cobra.Command{
		Use:     "sa",
		Short:   "Commands for working with kubernetes ServiceAccount resources",
		Aliases: []string{"serviceaccount", "serviceaccounts"},
		Run: func(command *cobra.Command, _ []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Error(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(secret.NewCmdServiceAccountSecrets()))
	return command
}
