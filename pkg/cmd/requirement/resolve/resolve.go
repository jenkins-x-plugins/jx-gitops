package resolve

import (
	"fmt"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
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
		Resolves any missing values in the jx-requirements.yml which can be detected.

For example if the provider is GKE then this step will automatically default the project, cluster name and location values if they are not in the 'jx-requirements.yml' file.
`)

	cmdExample = templates.Examples(`
		%s requirements resolve 
	`)
)

// Options the options for the command
type Options struct {
	Dir                  string
	NoCommit             bool
	NoInClusterCheck     bool
	CommandRunner        cmdrunner.CommandRunner
	Namespace            string
	SecretName           string
	GKEConfig            GKEConfig
	KubeClient           kubernetes.Interface
	gitClient            gitclient.Interface
	requirements         *config.RequirementsConfig
	requirementsFileName string
}

// NewCmdRequirementsResolve creates a command object for the command
func NewCmdRequirementsResolve() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "resolve",
		Short:   "Resolves any missing values in the jx-requirements.yml which can be detected",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to run the git push command from")
	cmd.Flags().BoolVarP(&o.NoCommit, "no-commit", "n", false, "disables performing a git commit if there are changes")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace used to find the git operator secret for the git repository if running in cluster. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.SecretName, "secret", "", "jx-boot", "the name of the Secret to find the git URL, username and password for creating a git credential if running inside the cluster")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	var err error
	o.requirements, o.requirementsFileName, err = config.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	if o.requirements == nil {
		return errors.Errorf("no 'jx-requirements.yml' file found in dir %s", o.Dir)
	}
	provider := o.requirements.Cluster.Provider
	if provider == "" {
		return errors.Errorf("missing kubernetes provider name at 'cluster.provider' in file: %s", o.requirementsFileName)
	}

	err = o.resolvePipelineUsername()
	if err != nil {
		return errors.Wrapf(err, "failed to resolve pipeilne user")
	}

	switch provider {
	case "gke":
		return o.ResolveGKE()
	default:
		log.Logger().Infof("no resolve logic for kubernetes provider %s", termcolor.ColorInfo(provider))
		return nil
	}
}

func (o *Options) GitClient() gitclient.Interface {
	if o.gitClient == nil {
		o.gitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.gitClient
}

func (o *Options) resolvePipelineUsername() error {
	if o.requirements.PipelineUser == nil {
		o.requirements.PipelineUser = &config.UserNameEmailConfig{}
	}
	if o.requirements.PipelineUser.Username != "" && o.requirements.PipelineUser.Email != "" {
		return nil
	}
	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}
	ns := o.Namespace
	name := o.SecretName
	secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Logger().Warnf("could not find secret %s in namespace %s", name, ns)
			return nil
		}
		return errors.Wrapf(err, "failed to find Secret %s in namespace %s", name, ns)
	}
	data := secret.Data
	username := ""
	email := ""
	if data != nil {
		username = string(data["username"])
		email = string(data["email"])
	}
	if username == "" {
		log.Logger().Warnf("no username in secret %s in namespace %s", name, ns)
	}
	if email == "" {
		log.Logger().Warnf("no email in secret %s in namespace %s", name, ns)
	}
	modified := false
	if o.requirements.PipelineUser.Username == "" && username != "" {
		o.requirements.PipelineUser.Username = username
		modified = true
	}
	if o.requirements.PipelineUser.Email == "" && email != "" {
		o.requirements.PipelineUser.Email = email
		modified = true
	}

	if !modified {
		return nil
	}
	err = o.requirements.SaveConfig(o.requirementsFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save modified requirements file %s", o.requirementsFileName)

	}
	log.Logger().Infof("modified the pipeline user in %s", termcolor.ColorInfo(o.requirementsFileName))
	return nil
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
