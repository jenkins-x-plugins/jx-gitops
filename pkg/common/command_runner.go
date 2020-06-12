package common

import (
	"github.com/jenkins-x/jx/pkg/util"
)

// CommandRunner represents a command runner so that it can be stubbed out for testing
type CommandRunner func(*util.Command) (string, error)

// DefaultCommandRunner default runner if none is set
func DefaultCommandRunner(c *util.Command) (string, error) {
	return c.RunWithoutRetry()
}
