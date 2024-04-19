package condition

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/filters"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Runs a command if the condition is true
`)

	cmdExample = templates.Examples(`
		# runs a command if the last commit messsage has a given prefix
		%s condition --last-commit-msg-prefix 'Merge pull request' -- make all commit push

you can use ! in front of a filter to be the equivalent of not matching the condition. e.g.

		# runs a command if the last commit message does not have a given prefix
		%s condition --last-commit-msg-prefix '!Merge pull request' -- make all commit push

	`)
)

// Options the options for the command
type Options struct {
	Dir                     string
	Args                    []string
	LastCommitMessageFilter filters.StringFilter
	BatchMode               bool
	CommandRunner           cmdrunner.CommandRunner
	Out                     io.Writer
	Err                     io.Writer
}

// NewCmdCondition creates a command object for the command
func NewCmdCondition() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "condition [flags] command arguments...",
		Short:   "Runs a command if the condition is true",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, args []string) {
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", "", "the directory to run the git push command from")

	o.LastCommitMessageFilter.AddFlags(cmd, "last-commit-msg", "last commit message")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if len(o.Args) == 0 {
		return errors.Errorf("no command or command arguments specified")
	}

	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}

	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "git",
		Args: []string{"log", "-1", "--pretty=%B"},
	}
	lastCommitMessage, err := o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run %s", c.CLI())
	}
	lastCommitMessage = strings.TrimSpace(lastCommitMessage)
	log.Logger().Infof("found last commit message: %s", termcolor.ColorStatus(lastCommitMessage))

	if o.LastCommitMessageFilter.Matches(lastCommitMessage) {
		if o.Out == nil {
			o.Out = os.Stdout
		}
		if o.Err == nil {
			o.Err = os.Stderr
		}
		c = &cmdrunner.Command{
			Dir:  o.Dir,
			Name: o.Args[0],
			Args: o.Args[1:],
			Out:  o.Out,
			Err:  o.Err,
		}
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run %s", c.CLI())
		}
	}
	return nil
}
