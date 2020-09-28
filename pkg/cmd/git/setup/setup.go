package setup

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/credentialhelper"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/pkg/kube"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Sets up git to ensure the git user name and email is setup.

This is typically used in a pipeline to ensure git can do commits.
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
	OutputFile           string
	DisableInClusterTest bool
	Namespace            string
	OperatorNamespace    string
	SecretName           string
	KubeClient           kubernetes.Interface
	CommandRunner        cmdrunner.CommandRunner
	gitClient            gitclient.Interface
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
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", "", "the directory to run the git push command from")
	cmd.Flags().StringVarP(&o.UserName, "name", "n", "", "the git user name to use if one is not setup")
	cmd.Flags().StringVarP(&o.UserEmail, "email", "e", "", "the git user email to use if one is not setup")
	cmd.Flags().StringVarP(&o.OutputFile, "credentials-file", "", "", "The destination of the git credentials file to generate. If not specified uses $XDG_CONFIG_HOME/git/credentials or $HOME/git/credentials")
	cmd.Flags().StringVarP(&o.OperatorNamespace, "operator-namespace", "", "jx-git-operator", "the namespace used by the git operator to find the secret for the git repository if running in cluster")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace used to find the git operator secret for the git repository if running in cluster. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.SecretName, "secret", "", "jx-boot", "the name of the Secret to find the git URL, username and password for creating a git credential if running inside the cluster")
	cmd.Flags().BoolVarP(&o.DisableInClusterTest, "fake-in-cluster", "", false, "for testing: lets you fake running this command inside a kubernetes cluster so that it can create the file: $XDG_CONFIG_HOME/git/credentials or $HOME/git/credentials")
	return cmd, o
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

	if o.DisableInClusterTest || IsInCluster() {
		outFile, err := o.determineOutputFile()
		if err != nil {
			return errors.Wrap(err, "unable to determine for git credentials")
		}

		return o.createGitCredentialsFile(outFile, credentials)
	}
	return nil
}

func (o *Options) GitClient() gitclient.Interface {
	if o.gitClient == nil {
		o.gitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.gitClient
}

// findCredentials detects the git operator secret so we have default credentials
func (o *Options) findCredentials() ([]credentialhelper.GitCredential, error) {
	var credentialList []credentialhelper.GitCredential

	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create kube client")
	}
	ns := o.Namespace
	name := o.SecretName
	secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if err != nil && o.OperatorNamespace != o.Namespace {
		var err2 error
		secret, err2 = o.KubeClient.CoreV1().Secrets(o.OperatorNamespace).Get(name, metav1.GetOptions{})
		if err2 == nil {
			err = nil
		}
	}
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Logger().Warnf("could not find secret %s in namespace %s", name, ns)
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to find Secret %s in namespace %s", name, ns)
	}
	data := secret.Data
	if data != nil {
		gitURL := string(data["url"])
		if gitURL == "" {
			log.Logger().Warnf("secret %s in namespace %s does not have a url entry", name, ns)
			return nil, nil
		}
		// lets convert the git URL into a provider URL
		gitInfo, err := giturl.ParseGitURL(gitURL)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse git URL %s", gitURL)
		}
		gitProviderURL := gitInfo.HostURL()

		username := string(data["username"])
		if o.UserName == "" {
			o.UserName = username
		}
		password := string(data["password"])
		credential, err := credentialhelper.CreateGitCredentialFromURL(gitProviderURL, username, password)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid git auth information")
		}
		credentialList = append(credentialList, credential)
	}
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

	if err := ioutil.WriteFile(fileName, data, files.DefaultDirWritePermissions); err != nil {
		return fmt.Errorf("failed to write to %s: %s", fileName, err)
	}
	log.Logger().Infof("Generated Git credentials file %s", termcolor.ColorInfo(fileName))
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
