package merge

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/gitlog"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Merge a number of SHAs into the HEAD of the main branch.

		This command merges a list of commits into a specified branch. If the branch does not exist among local branches, then
		it is first created.

		If both --pull-refs and --sha flags are specified then only those commits specified by --sha are merged into the
		base branch.

		If --include-comment or --exclude-comment flags are specified, then --pull-number flag needs to be set as well.
		If only one of --include-comment or --exclude-comment, then only that one is used to filter commits while other is
		ignored. If both are specified, then only those commits which satisfy --include-comment and do not satisfy the
		--exclude-comment regex are added. Only those commits which are reachable by from pull request and are not reachable
		by base branch are included to be merged into the base branch.
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
	GitMergeArgs         []string
	Remote               string
	Dir                  string
	BaseBranch           string
	BaseSHA              string
	PullRefs             string
	PullNumber           string
	IncludeCommitComment string
	ExcludeCommitComment string
	Rebase               bool
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
		"if not specified then the value of the env var $PULL_REFS is parsed")
	cmd.Flags().StringVarP(&o.Remote, "remote", "", "origin", "The name of the remote")
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "The directory in which the git repo is checked out")
	cmd.Flags().StringVarP(&o.BaseBranch, "base-branch", "", "", "The branch to merge to. If not specified then either $PULL_BASE_REF is used or the first entry in $PULL_REFS is used ")
	cmd.Flags().StringVarP(&o.BaseSHA, "base-sha", "", "", "The SHA to use on the base branch. Iff not specified then $PULL_BASE_SHA is used or the first entry in $PULL_REFS is used")
	cmd.Flags().StringVarP(&o.PullRefs, "pull-refs", "", "", "The PullRefs to parse")
	cmd.Flags().StringVarP(&o.PullNumber, "pull-number", "", "", "The Pull Request number to use when filtering commits to merge")

	cmd.Flags().StringVarP(&o.UserName, "name", "n", "", "the git user name to use if one is not setup")
	cmd.Flags().StringVarP(&o.UserEmail, "email", "e", "", "the git user email to use if one is not setup")

	cmd.Flags().StringVarP(&o.IncludeCommitComment, "include-comment", "", "", "the regular expression to filter commit comment to include in the merge")
	cmd.Flags().StringVarP(&o.ExcludeCommitComment, "exclude-comment", "", "", "the regular expression to filter commit comment to exclude in the merge")
	cmd.Flags().StringArrayVarP(&o.GitMergeArgs, "merge-arg", "", nil, "the extra arguments to pass to the 'git merge $sha' command to perform the merge")

	cmd.Flags().BoolVarP(&o.Rebase, "rebase", "r", false, "use git rebase instead of merge")

	cmd.Flags().BoolVarP(&o.DisableInClusterTest, "fake-in-cluster", "", false, "for testing: lets you fake running this command inside a kubernetes cluster so that it can create the file: $XDG_CONFIG_HOME/git/credentials or $HOME/git/credentials")

	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	if o.BaseBranch == "" {
		o.BaseBranch = os.Getenv("PULL_BASE_REF")
	}
	if o.BaseSHA == "" {
		o.BaseSHA = os.Getenv("PULL_BASE_SHA")
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
				for _, p := range pullRefs.ToMerge {
					o.SHAs = append(o.SHAs, p.SHA)
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

	if o.BaseSHA != "" && o.Rebase {
		return o.RebaseToBaseSHA()
	}
	if o.IncludeCommitComment != "" || o.ExcludeCommitComment != "" {
		o.SHAs, err = o.FindCommitsToMerge()
		if err != nil {
			return errors.Wrapf(err, "failed to find commit titles for include: %s exclude %s", o.IncludeCommitComment, o.ExcludeCommitComment)
		}
	}

	if len(o.SHAs) == 0 {
		log.Logger().Warnf("no SHAs to merge, falling back to initial cloned commit")
		return nil
	}

	err = FetchAndMergeSHAs(o.GitClient, o.SHAs, o.BaseBranch, o.BaseSHA, o.Remote, o.Dir, o.GitMergeArgs)
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

func (o *Options) RebaseToBaseSHA() error {
	args := []string{"rebase"}
	for _, a := range o.GitMergeArgs {
		args = append(args, a)
	}
	sha := o.BaseSHA
	args = append(args, sha)

	dir := o.Dir
	_, err := o.GitClient.Command(dir, args...)
	if err != nil {
		return errors.Wrapf(err, "rebasing %s into master", sha)
	}
	log.Logger().Infof("rebased git to %s", sha)
	log.Logger().Debugf("ran: git rebase %s", strings.Join(args, " "))
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

func (o *Options) FindCommitsToMerge() ([]string, error) {
	includeRE, err := ToRegexOrNil(o.IncludeCommitComment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create include commit title regex")
	}
	exludeRE, err := ToRegexOrNil(o.ExcludeCommitComment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create exclude commit title regex")
	}

	if o.PullNumber == "" {
		o.PullNumber = os.Getenv("PULL_NUMBER")
	}
	if o.BaseBranch == "" {
		return nil, errors.Errorf("no $PULL_BASE_REF specified")
	}
	if o.PullNumber == "" {
		return nil, errors.Errorf("no $PULL_NUMBER defined so cannot filter the commits on the pull request")
	}
	branchName := "tmp-pr-" + o.PullNumber

	_, err = o.GitClient.Command(o.Dir, "fetch", "origin", fmt.Sprintf("pull/%s/head:%s", o.PullNumber, branchName))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch pull request into temporary branch")
	}

	output, err := o.GitClient.Command(o.Dir, "--no-pager", "log", o.BaseBranch+".."+branchName, "--reverse", "--decorate=no", "--no-color")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the git log from the PR temporary branch %s", branchName)
	}
	commits := gitlog.ParseGitLog(output)
	if len(commits) == 0 {
		return nil, errors.Errorf("no git commits parsed for output: %s", output)
	}

	commits = filterCommits(commits, includeRE, exludeRE)
	if len(commits) == 0 {
		return nil, errors.Errorf("no git commits matched the commit title filters include: '%s', exclude: '%s'", o.IncludeCommitComment, o.ExcludeCommitComment)
	}

	var shas []string
	for _, commit := range commits {
		if commit.SHA != "" {
			shas = append(shas, commit.SHA)
		}
	}
	return shas, nil
}

func filterCommits(commits []*gitlog.Commit, includeRE *regexp.Regexp, excludeRE *regexp.Regexp) []*gitlog.Commit {
	var answer []*gitlog.Commit
	for _, commit := range commits {
		comment := commit.Comment
		if includeRE != nil && !includeRE.MatchString(comment) {
			continue
		}
		if excludeRE != nil && excludeRE.MatchString(comment) {
			continue
		}
		answer = append(answer, commit)
	}
	return answer
}

// ToRegexOrNil returns a regex for non-blank strings or returns an error if it cannot be parsed
func ToRegexOrNil(text string) (*regexp.Regexp, error) {
	if text == "" {
		return nil, nil
	}
	answer, err := regexp.Compile(text)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse regex %s", text)
	}
	return answer, nil
}
