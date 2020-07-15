package push

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/giturl"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Pushes the current git directory to the branch used to create the Pull Request
`)

	cmdExample = templates.Examples(`
		# pushes the current directories git contents to the branch used to create the current PR via $BRANCH_NAME
		%s pr push 
	`)

	pathSeparator = string(os.PathSeparator)
)

// KptOptions the options for the command
type Options struct {
	Dir           string
	Branch        string
	Repository    string
	SourceURL     string
	GitServerURL  string
	GitKind       string
	GitToken      string
	Number        int
	CommandRunner cmdrunner.CommandRunner
	ScmClient     *scm.Client
}

// NewCmdPullRequestPush creates a command object for the command
func NewCmdPullRequestPush() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "push",
		Short:   "Pushes the current git directory to the branch used to create the Pull Request",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", "", "the directory to run the git push command from")
	cmd.Flags().StringVarP(&o.SourceURL, "source", "s", "", "the git source URL of the current git clone")
	cmd.Flags().StringVarP(&o.Repository, "repo", "r", "", "the full git repository name of the form 'owner/name' for the Pull Request")
	cmd.Flags().StringVarP(&o.Branch, " branch", "b", "", "the git branch to push to. If not specified we will find the branch from the PullRequest.Source property")
	cmd.Flags().StringVarP(&o.GitServerURL, " git-server", "", "", "the git server URL to create the git provider client. If not specified its defaulted from the current source URL")
	cmd.Flags().StringVarP(&o.GitKind, " git-kind", "", "", "the kind of git server to connect to")
	cmd.Flags().StringVarP(&o.GitToken, " git-token", "", "", "the git oauth token used to query the Pull Request to discover the branch name")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	repo, err := o.discoverGitServerURLAndRepository()
	if err != nil {
		return errors.Wrapf(err, "failed to discover git server URL")
	}
	if o.Repository == "" {
		o.Repository = repo
		if o.Repository == "" {
			return errors.Errorf("could not to discover the Repository name. Consider specifying the --repo option")
		}
	}
	if o.Number == 0 {
		o.Number, err = o.discoverPullRequest()
		if err != nil {
			return errors.Wrapf(err, "failed to discover the Pull Request number. Consider specifying the --number option")
		}
		if o.Number <= 0 {
			return errors.Errorf("could not to discover the Pull Request number. Consider specifying the --number option")
		}
	}
	if o.Branch == "" {
		o.Branch, err = o.discoverPullRequestBranch()
		if err != nil {
			return errors.Wrapf(err, "failed to discover the pull request branch. Consider specifying the --branch option")
		}
		if o.Branch == "" {
			return errors.Errorf("could not find branch fpr PR %d in repo %s", o.Number, o.Repository)
		}
	}
	return o.pushToBranch()
}

func (o *Options) pushToBranch() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	argSlices := [][]string{
		{
			"pull",
		},
		{
			"checkout", "-b", o.Branch,
		},
		{
			"push", "origin", o.Branch,
		},
	}

	for _, args := range argSlices {
		c := &cmdrunner.Command{
			Dir:  o.Dir,
			Name: "git",
			Args: args,
		}
		_, err := o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run command %s", c.CLI())
		}
	}
	return nil
}

func (o *Options) discoverGitServerURLAndRepository() (string, error) {
	if o.SourceURL == "" {
		o.SourceURL = os.Getenv("SOURCE_URL")
	}
	if o.SourceURL == "" {
		// lets try find the git URL from the current git clone
		var err error
		o.SourceURL, err = gitdiscovery.FindGitURLFromDir(o.Dir)
		if err != nil {
			return "", errors.Wrapf(err, "failed to discover git URL in dir %s. you could try pass the git URL as an argument", o.Dir)
		}
	}
	if o.SourceURL != "" {
		gitInfo, err := giturl.ParseGitURL(o.SourceURL)
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse git URL %s", o.SourceURL)
		}
		if o.GitServerURL == "" {
			o.GitServerURL = gitInfo.HostURL()
		}
		return scm.Join(gitInfo.Organisation, gitInfo.Name), nil
	}
	if o.SourceURL == "" {
		owner := os.Getenv("REPO_OWNER")
		repo := os.Getenv("REPO_NAME")
		if owner != "" && repo != "" {
			return scm.Join(owner, repo), nil
		}
	}
	return "", nil
}

func (o *Options) discoverPullRequest() (int, error) {
	branchName := strings.ToUpper(os.Getenv("BRANCH_NAME"))
	prPrefix := "PR-"
	if strings.HasPrefix(branchName, prPrefix) {
		prefix := strings.TrimPrefix(branchName, prPrefix)
		if prefix != "" {
			n, err := strconv.Atoi(prefix)
			if err != nil {
				return n, errors.Wrapf(err, "failed to parse %s from $BRANCH_NAME", prefix)
			}
			return n, nil
		}
	}
	return 0, nil
}

func (o *Options) discoverPullRequestBranch() (string, error) {
	if o.ScmClient == nil {
		var err error
		if o.GitServerURL == "" {
			return "", errors.Errorf("could not deduce the git server URL. Try specifying --source")
		}
		oauthToken, err := o.discoverGitToken()
		if err != nil {
			return "", errors.Wrapf(err, "failed to discover git auth token")
		}
		if oauthToken == "" {
			return "", errors.Errorf("could not deduce the git auth token. Try specifying --git-token")
		}
		o.ScmClient, err = factory.NewClient(o.GitKind, o.GitServerURL, oauthToken)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create Scm client")
		}
	}
	ctx := context.Background()
	pr, _, err := o.ScmClient.PullRequests.Find(ctx, o.Repository, o.Number)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find PR %d in repo %s", o.Number, o.Repository)
	}
	return pr.Source, nil
}

func (o *Options) discoverGitToken() (string, error) {
	oauthToken := o.GitToken
	if oauthToken == "" {
		oauthToken = os.Getenv("GIT_TOKEN")
	}
	if oauthToken == "" {
		// TODO discover via secret...
	}
	return oauthToken, nil
}
