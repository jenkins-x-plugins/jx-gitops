package get

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx-gitops/pkg/authhelpers"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx/v2/pkg/auth"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Gets a pull request and displays fields from it
`)

	cmdExample = templates.Examples(`
		# display the head source URL
		%s pr get --head-url 
	`)

	pathSeparator = string(os.PathSeparator)
)

// KptOptions the options for the command
type Options struct {
	Dir               string
	Repository        string
	SourceURL         string
	GitServerURL      string
	GitKind           string
	GitToken          string
	UserName          string
	UserEmail         string
	Number            int
	ShowHeadURL       bool
	BatchMode         bool
	UseGitHubOAuth    bool
	CommandRunner     cmdrunner.CommandRunner
	ScmClient         *scm.Client
	AuthConfigService auth.ConfigService
	IOFileHandles     *util.IOFileHandles
	Result            *scm.PullRequest
	gitter            gits.Gitter
	gitClient         gitclient.Interface
}

// NewCmdPullRequestPush creates a command object for the command
func NewCmdPullRequestGet() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Gets a pull request and displays fields from it",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Repository, "repo", "r", "", "the full git repository name of the form 'owner/name' for the Pull Request")
	cmd.Flags().IntVarP(&o.Number, "pr", "", 0, "the Pull Request number. If not specified we will use $BRANCH_NAME")
	cmd.Flags().StringVarP(&o.GitServerURL, "git-server", "", "", "the git server URL to create the git provider client. If not specified its defaulted from the current source URL")
	cmd.Flags().StringVarP(&o.GitKind, "git-kind", "", "", "the kind of git server to connect to")
	cmd.Flags().StringVarP(&o.GitToken, "git-token", "", "", "the git oauth token used to query the Pull Request to discover the branch name")
	cmd.Flags().BoolVarP(&o.ShowHeadURL, "head-url", "", false, "show the head clone URL of the PR")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	o.BatchMode = true
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
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
		o.Number, err = o.discoverPullRequestNumber()
		if err != nil {
			return errors.Wrapf(err, "failed to discover the Pull Request number. Consider specifying the --number option")
		}
		if o.Number <= 0 {
			return errors.Errorf("could not to discover the Pull Request number. Consider specifying the --number option")
		}
	}
	pr, err := o.discoverPullRequest()
	if err != nil {
		return errors.Wrapf(err, "failed to discover the pull request")
	}
	if pr == nil {
		return errors.Errorf("no Pull Request could be found for %d in repository %s", o.Number, o.Repository)
	}
	return o.displayPullRequest(pr)
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

func (o *Options) discoverPullRequestNumber() (int, error) {
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

func (o *Options) discoverPullRequest() (*scm.PullRequest, error) {
	if o.ScmClient == nil {
		var err error
		if o.GitServerURL == "" {
			return nil, errors.Errorf("could not deduce the git server URL. Try specifying --source")
		}
		oauthToken, err := o.discoverGitToken()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to discover git auth token")
		}
		if oauthToken != "" {
			o.ScmClient, err = factory.NewClient(o.GitKind, o.GitServerURL, oauthToken)
		} else {
			o.ScmClient, _, err = o.createScmClient(o.GitServerURL, "", o.GitKind)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create Scm client")
		}
	}
	ctx := context.Background()
	pr, _, err := o.ScmClient.PullRequests.Find(ctx, o.Repository, o.Number)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find PR %d in repo %s", o.Number, o.Repository)
	}
	return pr, nil
}

func (o *Options) discoverGitToken() (string, error) {
	oauthToken := o.GitToken
	if oauthToken == "" {
		oauthToken = os.Getenv("GIT_TOKEN")
	}
	if oauthToken == "" {
		// TODO discover via secret?...
		return "", options.MissingOption("git-token")
	}
	return oauthToken, nil
}

// CreateScmClient creates a new scm client
func (o *Options) createScmClient(gitServer, owner, gitKind string) (*scm.Client, string, error) {
	if IsInCluster() {
		err := o.InitGitConfigAndUser()
		if err != nil {
			return nil, "", errors.Wrapf(err, "failed to init git")
		}
	}
	af, err := authhelpers.NewAuthFacadeWithArgs(o.AuthConfigService, o.Git(), o.IOFileHandles, o.BatchMode, o.UseGitHubOAuth)
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to create git auth facade")
	}
	scmClient, token, _, err := af.ScmClient(gitServer, owner, gitKind)
	if err != nil {
		return scmClient, token, errors.Wrapf(err, "failed to create SCM client for server %s", gitServer)
	}
	return scmClient, token, nil
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}

func (o *Options) InitGitConfigAndUser() error {
	gitClient := o.GitClient()
	_, _, err := gitclient.EnsureUserAndEmailSetup(gitClient, o.Dir, o.UserName, o.UserEmail)
	if err != nil {
		return errors.Wrapf(err, "failed to setup git user and email")
	}
	err = gitclient.SetCredentialHelper(gitClient, "")
	if err != nil {
		return errors.Wrapf(err, "failed to setup credential store")
	}
	return nil
}

func (o *Options) Git() gits.Gitter {
	if o.gitter == nil {
		o.gitter = gits.NewGitCLI()
	}
	return o.gitter
}
func (o *Options) GitClient() gitclient.Interface {
	if o.gitClient == nil {
		o.gitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.gitClient
}

func (o *Options) displayPullRequest(pr *scm.PullRequest) error {
	o.Result = pr

	if o.ShowHeadURL {
		log.Logger().Info(pr.Head.Repo.Clone)
		return nil
	}

	data, err := yaml.Marshal(pr)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal PullRequest as YAML")
	}
	log.Logger().Info(string(data))
	return nil

}
