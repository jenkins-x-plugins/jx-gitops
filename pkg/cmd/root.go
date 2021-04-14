package cmd

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/annotate"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/apply"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/condition"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/copy"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/gc"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/git"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/hash"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/image"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/ingress"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/jenkins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kpt"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kustomize"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/label"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/lint"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/namespace"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/plugin"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/postprocess"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/pr"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/rename"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/requirement"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/sa"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/scheduler"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/upgrade"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/variables"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/version"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/versionstream"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/webhook"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootcmd.TopLevelCommand,
		Short: "commands for working with GitOps based git repositories",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	cmd.AddCommand(helm.NewCmdHelm())
	cmd.AddCommand(helmfile.NewCmdHelmfile())
	cmd.AddCommand(gc.NewCmdGC())
	cmd.AddCommand(git.NewCmdGit())
	cmd.AddCommand(jenkins.NewCmdJenkins())
	cmd.AddCommand(kpt.NewCmdKpt())
	cmd.AddCommand(plugin.NewCmdPlugin())
	cmd.AddCommand(pr.NewCmdPR())
	cmd.AddCommand(requirement.NewCmdRequirement())
	cmd.AddCommand(repository.NewCmdRepository())
	cmd.AddCommand(sa.NewCmdServiceAccount())
	cmd.AddCommand(webhook.NewCmdWebhook())

	cmd.AddCommand(cobras.SplitCommand(annotate.NewCmdUpdateAnnotate()))
	cmd.AddCommand(cobras.SplitCommand(apply.NewCmdApply()))
	cmd.AddCommand(cobras.SplitCommand(condition.NewCmdCondition()))
	cmd.AddCommand(cobras.SplitCommand(copy.NewCmdCopy()))
	cmd.AddCommand(cobras.SplitCommand(hash.NewCmdHashAnnotate()))
	cmd.AddCommand(cobras.SplitCommand(image.NewCmdUpdateImage()))
	cmd.AddCommand(cobras.SplitCommand(ingress.NewCmdUpdateIngress()))
	cmd.AddCommand(cobras.SplitCommand(kustomize.NewCmdKustomize()))
	cmd.AddCommand(cobras.SplitCommand(label.NewCmdUpdateLabel()))
	cmd.AddCommand(cobras.SplitCommand(lint.NewCmdLint()))
	cmd.AddCommand(cobras.SplitCommand(namespace.NewCmdUpdateNamespace()))
	cmd.AddCommand(cobras.SplitCommand(rename.NewCmdRename()))
	cmd.AddCommand(cobras.SplitCommand(postprocess.NewCmdPostProcess()))
	cmd.AddCommand(cobras.SplitCommand(scheduler.NewCmdScheduler()))
	cmd.AddCommand(cobras.SplitCommand(split.NewCmdSplit()))
	cmd.AddCommand(cobras.SplitCommand(upgrade.NewCmdUpgrade()))
	cmd.AddCommand(cobras.SplitCommand(variables.NewCmdVariables()))
	cmd.AddCommand(cobras.SplitCommand(version.NewCmdVersion()))
	cmd.AddCommand(cobras.SplitCommand(versionstream.NewCmdVersionstream()))
	return cmd
}
