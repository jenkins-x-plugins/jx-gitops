package common

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
)

// CommandRunner represents a command runner so that it can be stubbed out for testing
type CommandRunner func(*util.Command) (string, error)

// DefaultCommandRunner default runner if none is set
func DefaultCommandRunner(c *util.Command) (string, error) {
	log.Logger().Infof("about to run %s in dir %s", util.ColorInfo(c.String()), util.ColorInfo(c.Dir))
	return c.RunWithoutRetry()
}

// DryRunCommandRunner output the commands to be run
func DryRunCommandRunner(c *util.Command) (string, error) {
	log.Logger().Infof(c.String())
	return "", nil
}
