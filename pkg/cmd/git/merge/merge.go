package merge

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Merge a number of SHAs into the HEAD of the main branch
`)

	cmdExample = templates.Examples(`
		%s git merge 
	`)

	info = termcolor.ColorInfo
)

// Options the options for the command
type Options struct {
	UserName             string
	UserEmail            string
	SHAs                 []string
	Remote               string
	Dir                  string
	BaseBranch           string
	BaseSHA              string
	PullRefs             string
	DisableInClusterTest bool

	CommandRunner cmdrunner.CommandRunner
	GitClient     gitclient.Interface
}

// NewCmdGitMerge creates a command object for the command
func NewCmdGitMerge() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "merge",
		Short:   "Merge a number of SHAs into the HEAD of the main branch",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringArrayVarP(&o.SHAs, "sha", "", make([]string, 0), "The SHA(s) to merge, "+
		"if not specified then the value of the env var PULL_REFS is used")
	cmd.Flags().StringVarP(&o.Remote, "remote", "", "origin", "The name of the remote")
	cmd.Flags().StringVarP(&o.Dir, "dir", "", "", "The directory in which the git repo is checked out")
	cmd.Flags().StringVarP(&o.BaseBranch, "baseBranch", "", "", "The branch to merge to, "+
		"if not specified then the  first entry in PULL_REFS is used ")
	cmd.Flags().StringVarP(&o.BaseSHA, "baseSHA", "", "", "The SHA to use on the base branch, "+
		"if not specified then the first entry in PULL_REFS is used")
	cmd.Flags().StringVarP(&o.PullRefs, "pull-refs", "", "", "The PullRefs to parse")

	cmd.Flags().StringVarP(&o.UserName, "name", "n", "", "the git user name to use if one is not setup")
	cmd.Flags().StringVarP(&o.UserEmail, "email", "e", "", "the git user email to use if one is not setup")
	cmd.Flags().BoolVarP(&o.DisableInClusterTest, "fake-in-cluster", "", false, "for testing: lets you fake running this command inside a kubernetes cluster so that it can create the file: $XDG_CONFIG_HOME/git/credentials or $HOME/git/credentials")

	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	if len(o.SHAs) == 0 || o.BaseBranch == "" || o.BaseSHA == "" {
		pullRefs := o.PullRefs
		if pullRefs == "" {
			pullRefs = os.Getenv("PULL_REFS")
		}
		// Try to look in the env vars
		if pullRefs != "" {
			log.Logger().Infof("Using SHAs from PULL_REFS=%s", pullRefs)
			pullRefs, err := ParsePullRefs(pullRefs)
			if err != nil {
				return errors.Wrapf(err, "parsing PULL_REFS=%s", pullRefs)
			}
			if len(o.SHAs) == 0 {
				o.SHAs = make([]string, 0)
				for _, sha := range pullRefs.ToMerge {
					o.SHAs = append(o.SHAs, sha)
				}
			}
			if o.BaseBranch == "" {
				o.BaseBranch = pullRefs.BaseBranch
			}
			if o.BaseSHA == "" {
				o.BaseSHA = pullRefs.BaseSha
			}
		}
	}
	if len(o.SHAs) == 0 {
		log.Logger().Warnf("no SHAs to merge, falling back to initial cloned commit")
		return nil
	}

	err = FetchAndMergeSHAs(o.GitClient, o.SHAs, o.BaseBranch, o.BaseSHA, o.Remote, o.Dir)
	if err != nil {
		return errors.Wrap(err, "error during merge")
	}

	/*
		if o.Verbose {
			commits, err := o.getMergedCommits()
			if err != nil {
				return errors.Wrap(err, "unable to write merge result")
			}
			o.logCommits(commits, o.BaseBranch)
		}
	*/

	return nil
}

// Validate validates the inputs are valid
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	_, _, err := gitclient.SetUserAndEmail(o.GitClient, o.Dir, o.UserName, o.UserEmail, o.DisableInClusterTest)
	if err != nil {
		return errors.Wrapf(err, "failed to setup git user and email")
	}
	return nil
}
