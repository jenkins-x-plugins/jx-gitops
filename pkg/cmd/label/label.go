package label

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/tagging"
	"github.com/spf13/cobra"
)

// NewCmdUpdateLabel creates a command object for the command annotate
func NewCmdUpdateLabel() (*cobra.Command, *tagging.Options) {
	return tagging.NewCmdUpdateTag("label", "label")
}
