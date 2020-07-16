package cmd

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/annotate"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/condition"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/extsecret"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/hash"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/ingress"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/jx_apps"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/kustomize"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/label"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/namespace"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/pr"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/repository"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/scheduler"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/version"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootcmd.TopLevelCommand,
		Short: "GitOps utility commands",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	cmd.AddCommand(helm.NewCmdHelm())
	cmd.AddCommand(jx_apps.NewCmdJxApps())
	cmd.AddCommand(kpt.NewCmdKpt())
	cmd.AddCommand(pr.NewCmdPR())
	cmd.AddCommand(cobras.SplitCommand(annotate.NewCmdUpdateAnnotate()))
	cmd.AddCommand(cobras.SplitCommand(extsecret.NewCmdExtSecrets()))
	cmd.AddCommand(cobras.SplitCommand(condition.NewCmdCondition()))
	cmd.AddCommand(cobras.SplitCommand(hash.NewCmdHashAnnotate()))
	cmd.AddCommand(cobras.SplitCommand(ingress.NewCmdUpdateIngress()))
	cmd.AddCommand(cobras.SplitCommand(kustomize.NewCmdKustomize()))
	cmd.AddCommand(cobras.SplitCommand(label.NewCmdUpdateLabel()))
	cmd.AddCommand(cobras.SplitCommand(namespace.NewCmdUpdateNamespace()))
	cmd.AddCommand(cobras.SplitCommand(repository.NewCmdUpdateRepository()))
	cmd.AddCommand(cobras.SplitCommand(scheduler.NewCmdScheduler()))
	cmd.AddCommand(cobras.SplitCommand(split.NewCmdSplit()))
	cmd.AddCommand(cobras.SplitCommand(version.NewCmdVersion()))
	return cmd
}
