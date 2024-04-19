package webhook

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/webhook/delete"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/webhook/update"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

var webHookUpdateRetriableErrors = []string{
	"dial tcp \\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}:\\d+: i/o timeout",
}

// NewCmdWebhook creates the new command
func NewCmdWebhook() *cobra.Command {
	command := &cobra.Command{
		Use:     "webhook",
		Short:   "Commands for working with WebHooks on your source repositories",
		Aliases: []string{"webhooks", "hook", "hooks"},
		Run: func(command *cobra.Command, _ []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(helper.RetryOnErrorCommand(cobras.SplitCommand(update.NewCmdWebHookVerify()), helper.RegexRetryFunction(webHookUpdateRetriableErrors)))
	command.AddCommand(helper.RetryOnErrorCommand(cobras.SplitCommand(delete.NewCmdWebHookDelete()), helper.RegexRetryFunction(webHookUpdateRetriableErrors)))
	return command
}
