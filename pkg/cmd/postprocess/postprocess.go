package postprocess

import (
	"context"
	"fmt"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultSecretName default secret name for the post process commands
	DefaultSecretName = "jx-post-process"

	// DefaultSecretNamespace default namespace for the post process commands
	DefaultSecretNamespace = "default"
)

var (
	cmdLong = templates.LongDesc(`
		Post processes kubernetes resources to enrich resources like ServiceAccounts with cloud specific sensitive data to enable IAM rles
`)

	cmdExample = templates.Examples(`
		# after applying the resources lets post process them
		%s postprocess

		# you can register some post processing commands, such as to annotate a ServiceAccount via:
		kubectl create secret generic jx-post-process -n default  --from-literal=commands="kubectl annotate sa tekton-bot hello=world"	
	`)
)

// Options the options for the command
type Options struct {
	Namespace     string
	SecretName    string
	Shell         string
	KubeClient    kubernetes.Interface
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdPostProcess creates a command object for the command
func NewCmdPostProcess() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "postprocess",
		Short:   "Post processes kubernetes resources to enrich resources like ServiceAccounts with cloud specific sensitive data to enable IAM rles",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", DefaultSecretNamespace, "the namespace to look for the post processing Secret")
	cmd.Flags().StringVarP(&o.SecretName, "secret", "s", DefaultSecretName, "the name of the Secret with the post process scripts to apply")
	cmd.Flags().StringVarP(&o.Shell, "shell", "", "sh", "the location of the shell binary to execute")
	return cmd, o
}

// Run runs the command
func (o *Options) Run() error {
	var err error
	o.KubeClient, err = kube.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client")
	}

	info := termcolor.ColorInfo
	ns := o.Namespace
	name := o.SecretName
	secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Logger().Infof("there is no post processing Secret %s in namespace %s so not performing any additional post processing steps", info(name), info(ns))
			return nil
		}
		return errors.Wrapf(err, "failed to load ")
	}
	commands := ""
	if secret != nil && secret.Data != nil {
		commands = string(secret.Data["commands"])
	}
	if commands == "" {
		log.Logger().Warnf("the post processing Secret %s in namespace %s has no 'commands' key so not performing any additional post processing steps", name, ns)
		return nil
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	c := &cmdrunner.Command{
		Name: o.Shell,
		Args: []string{"-c", commands},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run command %s", c.CLI())
	}
	log.Logger().Infof("invoked the post processing commands from Secret %s in namespace %s", info(name), info(ns))
	return nil
}
