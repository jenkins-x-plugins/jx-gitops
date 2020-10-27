package pr

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/pr/comment"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/pr/get"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/pr/label"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/pr/push"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdPR creates the new command
func NewCmdPR() *cobra.Command {
	command := &cobra.Command{
		Use:     "pr",
		Short:   "Commands for working with Pull Requests",
		Aliases: []string{"pullrequest", "pullrequests"},
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(comment.NewCmdPullRequestComment()))
	command.AddCommand(cobras.SplitCommand(get.NewCmdPullRequestGet()))
	command.AddCommand(cobras.SplitCommand(label.NewCmdPullRequestLabel()))
	command.AddCommand(cobras.SplitCommand(push.NewCmdPullRequestPush()))
	return command
}
