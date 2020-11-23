package clone

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/setup"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Clones the cluster git repository using the URL, git user and token from the Secret

`)

	cmdExample = templates.Examples(`
		%s git clone 
	`)
)

// Options the options for the command
type Options struct {
	setup.Options
	CloneDir string
}

// NewCmdGitClone creates a command object for the command
func NewCmdGitClone() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "clone",
		Short:   "Clones the cluster git repository using the URL, git user and token from the Secret",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.CloneDir, "clone-dir", "", "", "the directory to clone the repository to")
	o.Options.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to setup git")
	}

	if o.CloneDir == "" {
		o.CloneDir = filepath.Join(o.Dir, "source")
	}
	u := o.GitURL
	if u == "" {
		return errors.Errorf("no git URL found in th eboot secret")
	}
	gitInitCommands := o.GitInitCommands
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	if gitInitCommands != "" {
		c := &cmdrunner.Command{
			Name: "sh",
			Args: []string{"-c", gitInitCommands},
			Out:  os.Stdout,
			Err:  os.Stderr,
		}
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run git init commands: %s", c.CLI())
		}
		log.Logger().Infof("ran git init commands: %s", gitInitCommands)
	}

	_, err = gitclient.CloneToDir(o.GitClient(), u, o.CloneDir)
	if err != nil {
		return errors.Wrapf(err, "failed to git clone URL %s to dir %s", u, o.CloneDir)
	}
	log.Logger().Infof("cloned repository %s to dir %s", info(u), info(o.CloneDir))
	return nil
}
