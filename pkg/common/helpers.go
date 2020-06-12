package common

import (
	"os"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// BinaryName the binary name to use in help docs
var BinaryName string

// TopLevelCommand the top level command name
var TopLevelCommand string

func init() {
	BinaryName = os.Getenv("BINARY_NAME")
	if BinaryName == "" {
		BinaryName = "jx-gitops"
	}
	TopLevelCommand = os.Getenv("TOP_LEVEL_COMMAND")
	if TopLevelCommand == "" {
		TopLevelCommand = "jx-gitops"
	}
}

// SplitCommand helper command to ignore the options object
func SplitCommand(cmd *cobra.Command, _ interface{}) *cobra.Command {
	return cmd
}

// GetIOFileHandles lazily creates a file handles object if the input is nil
func GetIOFileHandles(h *util.IOFileHandles) util.IOFileHandles {
	if h == nil {
		h = &util.IOFileHandles{
			Err: os.Stderr,
			In:  os.Stdin,
			Out: os.Stdout,
		}
	}
	return *h
}
