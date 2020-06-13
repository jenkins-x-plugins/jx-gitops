package cmd

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/annotate"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/ingress"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/label"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/namespace"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/version"
	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   common.TopLevelCommand,
		Short: "GitOps utility commands",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	cmd.AddCommand(helm.NewCmdHelm())
	cmd.AddCommand(common.SplitCommand(annotate.NewCmdUpdateAnnotate()))
	cmd.AddCommand(common.SplitCommand(ingress.NewCmdUpdateIngress()))
	cmd.AddCommand(common.SplitCommand(kpt.NewCmdUpdateKpt()))
	cmd.AddCommand(common.SplitCommand(label.NewCmdUpdateLabel()))
	cmd.AddCommand(common.SplitCommand(namespace.NewCmdUpdateNamespace()))
	cmd.AddCommand(common.SplitCommand(version.NewCmdVersion()))
	return cmd
}
