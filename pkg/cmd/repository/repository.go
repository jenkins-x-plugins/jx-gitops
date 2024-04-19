package repository

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/add"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/create"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/deletecmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/export"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/resolve"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdRepository creates the new command
func NewCmdRepository() *cobra.Command {
	command := &cobra.Command{
		Use:     "repository",
		Short:   "Commands for working with source repositories",
		Aliases: []string{"repo", "repos", "repositories"},
		Run: func(command *cobra.Command, _ []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(add.NewCmdAddRepository()))
	command.AddCommand(cobras.SplitCommand(create.NewCmdCreateRepository()))
	command.AddCommand(cobras.SplitCommand(deletecmd.NewCmdDeleteRepository()))
	command.AddCommand(cobras.SplitCommand(export.NewCmdExportConfig()))
	command.AddCommand(cobras.SplitCommand(resolve.NewCmdResolveRepository()))
	return command
}
