package annotate

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/tagging"
	"github.com/spf13/cobra"
)

// NewCmdUpdateAnnotate creates a command object for the command annotate
func NewCmdUpdateAnnotate() (*cobra.Command, *tagging.Options) {
	return tagging.NewCmdUpdateTag("annotate", "annotation")
}
