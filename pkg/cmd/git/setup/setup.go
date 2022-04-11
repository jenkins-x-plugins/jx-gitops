package setup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/boot"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/credentialhelper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Sets up git to ensure the git user name and email is setup. This is typically used in a pipeline to ensure git can do commits.

		The jx gitops git setup command ensures that we can authenticate with configured Git server by configuring the local
		credentials file in the home directory. This command tries to ensure the following things:

		- The user can be authenticated with Git provider (for example Github)
		- An email is associated with each automated commit message

		These credentials are written to ${HOME}/git/credentials file, where the ${HOME} directory is determined as:

		- value stored in XGD_CONFIG_HOME environment variable or
		- value stored in HOME environment variable or
		- value stored in USERPROFILE environment variable or
		- as current directory ./

		The credentials are determined by reading out the jx-requirements.yaml from the cluster repository and jx-boot
		Secret resource provisioned together with jx-git-operator in your Kubernetes namespace.

		The Git username and email are preferentially determined from PipelineUser field from jx-requirements.yaml, but if
		they are not available there then default email address jenkins-x@googlegroups.com is used . If the username could not be determined
		from jx-requirements.yaml, then it is determined from:

		- GIT_USERNAME environment variable or
		- GITHUB_ACTOR environment variable
		- In case that we are running in Kubernetes cluster from "username" field of the "jx-boot" Secret provisioned with
		"jx-git-operator"

		The password for Github user (or a token for the robot account, depending on which you configued) is determined in
		similar fashion. Namely the token is first determined from environment variable GITHUB_TOKEN, but if that fails, then
		further determination is dependent on execution environment of the command. Namely if it is running within Github
		actions, then the GITHUB_TOKEN environment variable is our last stop. Otherwise if the command is executed within
		Kubernetes cluster, then the secret is determined by reading the password field of the "jx-boot" Secret provisioned
		with the "jx-git-operator".

`)

	cmdExample = templates.Examples(`
		%s git setup 
	`)
)

// Options the options for the command
type Options struct {
	Dir                  string
	UserName             string
	UserEmail            string
	Password             string
	OutputFile           string
	Namespace            string
	OperatorNamespace    string
	SecretName           string
	GitURL               string
	GitProviderURL       string
	GitInitCommands      string
	DisableInClusterTest bool
	KubeClient           kubernetes.Interface
	CommandRunner        cmdrunner.CommandRunner
	gitClient            gitclient.Interface
	Requirements         *jxcore.Requirements
}

// NewCmdGitSetup creates a command object for the command
func NewCmdGitSetup() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "setup",
		Short:   "Sets up git to ensure the git user name and email is setup",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.AddFlags(cmd)
	return cmd, o
}

// AddFlags adds the command line flags
func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", "", "the directory to run the git setup command from")
	cmd.Flags().StringVarP(&o.UserName, "name", "n", "", "the git user name to use if one is not setup")
	cmd.Flags().StringVarP(&o.Password, "password", "", "", "the git password/token to use. if not specified it is detected from the git operator Secret")
	cmd.Flags().StringVarP(&o.UserEmail, "email", "e", "", "the git user email to use if one is not setup. Default value is jenkins-x@googlegroups.com, if none other is provided")
	cmd.Flags().StringVarP(&o.GitProviderURL, "git-provider", "", "", "the git provider URL. If not specified its detected from the git operator jx-boot Secret or defaults to https://github.com")
	cmd.Flags().StringVarP(&o.OutputFile, "credentials-file", "", "", "The destination of the git credentials file to generate. If not specified uses $XDG_CONFIG_HOME/git/credentials or $HOME/git/credentials")
	cmd.Flags().StringVarP(&o.OperatorNamespace, "operator-namespace", "", "jx-git-operator", "the namespace used by the git operator to find the secret for the git repository if running in cluster")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace used to find the git operator secret for the git repository if running in cluster. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.SecretName, "secret", "", "jx-boot", "the name of the Secret to find the git URL, username and password for creating a git credential if running inside the cluster")
	cmd.Flags().BoolVarP(&o.DisableInClusterTest, "fake-in-cluster", "", false, "for testing: lets you fake running this command inside a kubernetes cluster so that it can create the file: $XDG_CONFIG_HOME/git/credentials or $HOME/git/credentials")
}

// Run implements the command
func (o *Options) Run() error {
	gitClient := o.GitClient()

	// lets make sure there's a git config home dir
	homeDir := GetConfigHome()
	err := os.MkdirAll(homeDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure git config home directory exists %s", homeDir)
	}

	err = o.findCredentialsFromRequirements()
	if err != nil {
		return errors.Wrapf(err, "failed to find credentials from requirements")
	}

	// lets fetch the credentials so we can default the UserName if its not specified
	credentials, err := o.findCredentials()
	if err != nil {
		return errors.Wrap(err, "creating git credentials")
	}

	_, _, err = gitclient.SetUserAndEmail(gitClient, o.Dir, o.UserName, o.UserEmail, o.DisableInClusterTest)
	if err != nil {
		return errors.Wrapf(err, "failed to setup git user and email")
	}

	err = gitclient.SetCredentialHelper(gitClient, "")
	if err != nil {
		return errors.Wrapf(err, "failed to setup credential store")
	}

	if o.DisableInClusterTest || InGitHubActions() || IsInCluster() {
		outFile, err := o.determineOutputFile()
		if err != nil {
			return errors.Wrap(err, "unable to determine for git credentials")
		}

		err = o.createGitCredentialsFile(outFile, credentials)
		if err != nil {
			return errors.Wrapf(err, "failed to generate credentials file %s", outFile)
		}
		log.Logger().Infof("generated Git credentials file: %s with username: %s email: %s", info(outFile), info(o.UserName), info(o.UserEmail))

	}
	return nil
}

// InGitHubActions returns true if we are running inside a github action
func InGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func (o *Options) GitClient() gitclient.Interface {
	if o.gitClient == nil {
		if o.CommandRunner == nil {
			o.CommandRunner = cmdrunner.QuietCommandRunner
		}
		o.gitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.gitClient
}

// findCredentials detects the git operator secret so we have default credentials
func (o *Options) findCredentials() ([]credentialhelper.GitCredential, error) {
	var credentialList []credentialhelper.GitCredential

	if o.UserName == "" {
		o.UserName = os.Getenv("GIT_USERNAME")
	}
	if o.UserName == "" {
		o.UserName = os.Getenv("GITHUB_ACTOR")
	}
	if o.Password == "" {
		o.Password = os.Getenv("GIT_TOKEN")
	}
	if o.Password == "" {
		o.Password = os.Getenv("GITHUB_TOKEN")
	}

	if (o.Password == "" || o.UserName == "") && !InGitHubActions() {
		var err error
		o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create kube client")
		}
		bootSecret, err := boot.LoadBootSecret(o.KubeClient, o.Namespace, o.OperatorNamespace, o.SecretName, o.UserName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load the boot secret")
		}
		if bootSecret == nil {
			return nil, errors.Errorf("failed to find the boot secret")
		}

		gitURL := bootSecret.URL
		if o.GitProviderURL == "" {
			o.GitProviderURL = bootSecret.GitProviderURL
		}
		if gitURL != "" && o.GitProviderURL == "" {
			// lets convert the git URL into a provider URL
			gitInfo, err := giturl.ParseGitURL(gitURL)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse git URL %s", gitURL)
			}
			o.GitProviderURL = gitInfo.HostURL()
		}
		o.GitURL = gitURL
		o.GitInitCommands = bootSecret.GitInitCommands

		if o.UserName == "" {
			o.UserName = bootSecret.Username
		}
		if o.Password == "" {
			o.Password = bootSecret.Password
		}
	}
	if o.GitProviderURL == "" {
		o.GitProviderURL = "https://github.com"
	}
	if o.UserName == "" {
		return nil, options.MissingOption("name")
	}
	if o.Password == "" {
		return nil, options.MissingOption("password")
	}
	credential, err := credentialhelper.CreateGitCredentialFromURL(o.GitProviderURL, o.UserName, o.Password)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid git auth information")
	}
	credentialList = append(credentialList, credential)
	return credentialList, nil
}

func (o *Options) determineOutputFile() (string, error) {
	outFile := o.OutputFile
	if outFile == "" {
		outFile = GitCredentialsFile()
	}

	dir, _ := filepath.Split(outFile)
	if dir != "" {
		err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
		if err != nil {
			return "", err
		}
	}
	return outFile, nil
}

// CreateGitCredentialsFileFromUsernameAndToken creates the git credentials into file using the provided username, token & url
func (o *Options) createGitCredentialsFile(fileName string, credentials []credentialhelper.GitCredential) error {
	data, err := o.GitCredentialsFileData(credentials)
	if err != nil {
		return errors.Wrap(err, "creating git credentials")
	}

	if err := os.WriteFile(fileName, data, files.DefaultDirWritePermissions); err != nil {
		return fmt.Errorf("failed to write to %s: %s", fileName, err)
	}
	return nil
}

// GitCredentialsFileData takes the given git credentials and writes them into a byte array.
func (o *Options) GitCredentialsFileData(credentials []credentialhelper.GitCredential) ([]byte, error) {
	var buffer bytes.Buffer
	for _, gitCredential := range credentials {
		u, err := gitCredential.URL()
		if err != nil {
			log.Logger().Warnf("Ignoring incomplete git credentials %q", gitCredential)
			continue
		}

		buffer.WriteString(u.String() + "\n")
		// Write the https protocol in case only https is set for completeness
		if u.Scheme == "http" {
			u.Scheme = "https"
			buffer.WriteString(u.String() + "\n")
		}
	}

	return buffer.Bytes(), nil
}

func (o *Options) findCredentialsFromRequirements() error {
	if o.Requirements == nil {
		if o.Dir == "" {
			o.Dir = "."
		}
		path := filepath.Join(o.Dir, jxcore.RequirementsConfigFileName)
		exists, err := files.FileExists(path)
		if err != nil {
			return errors.Wrapf(err, "failed to check for file %s", path)
		}
		if !exists {
			log.Logger().Warnf("could not default pipeline user/email from requirements as file does not exist: %s", path)
			return nil
		}
		o.Requirements, _, err = jxcore.LoadRequirementsConfig(o.Dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load jx-requirements.yml file")
		}
	}
	if o.Requirements == nil {
		return nil
	}
	pu := o.Requirements.Spec.PipelineUser
	if pu == nil {
		return nil
	}
	if o.UserName == "" {
		o.UserName = pu.Username
		log.Logger().Infof("found git user.name %s from requirements", info(o.UserName))
	}
	if o.UserEmail == "" {
		o.UserEmail = pu.Email
		log.Logger().Infof("found git user.email %s from requirements", info(o.UserEmail))
	}
	return nil
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}

// GitCredentialsFile returns the location of the git credentials file
func GitCredentialsFile() string {
	cfgHome := GetConfigHome()
	return filepath.Join(cfgHome, "git", "credentials")
}

// GetConfigHome returns the home dir
func GetConfigHome() string {
	cfgHome := os.Getenv("XDG_CONFIG_HOME")
	if cfgHome == "" {
		cfgHome = homedir.HomeDir()
	}
	if cfgHome == "" {
		cfgHome = "."
	}
	return cfgHome
}
