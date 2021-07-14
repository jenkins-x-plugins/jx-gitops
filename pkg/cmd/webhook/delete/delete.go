package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Options the flags for updating webhooks
type Options struct {
	options.BaseOptions

	ScmClientFactory scmhelpers.Factory
	Org              string
	User             string
	Repo             string
	Filter           string
	AllWebhooks      bool
	DryRun           bool
	WarnOnFail       bool
	Namespace        string
	KubeClient       kubernetes.Interface
	JXClient         jxc.Interface
}

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Deletes the webhooks for all the source repositories optionally filtering by owner and/or repository

`)

	cmdExample = templates.Examples(`
		# delete all the webhooks for all SourceRepository and Environment resource:
		%s delete --filter https://foo.bar

		# only delete the webhooks for a given owner
		%s delete --owner=mycorp --filter https://foo.bar

		# delete all webhooks within an organisation
		%s delete --owner=mycorp --all-webhooks
`)
)

func NewCmdWebHookDelete() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "deletes the webhooks for all the source repositories optionally filtering by owner and/or repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName, rootcmd.BinaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Org, "owner", "o", "", "The name of the git organisation or user to filter on")
	cmd.Flags().StringVarP(&o.Repo, "repo", "r", "", "The name of the repository to filter on")
	cmd.Flags().StringVarP(&o.Filter, "filter", "", "", "The filter to match the endpoints to delete")
	cmd.Flags().BoolVarP(&o.AllWebhooks, "all-webhooks", "", false, "WARNING: will delete all webhooks from your source repositories. Do not use lightly.")
	cmd.Flags().BoolVarP(&o.WarnOnFail, "warn-on-fail", "", false, "If enabled lets just log a warning that we could not update the webhook")
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "", true, "If enabled doesn't actually delete any webhooks, just tells you what it will delete")

	o.ScmClientFactory.AddFlags(cmd)
	o.BaseOptions.AddBaseFlags(cmd)

	return cmd, o
}

// Validate verifies things are setup correctly
func (o *Options) Validate() error {
	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}
	o.JXClient, err = jxclient.LazyCreateJXClient(o.JXClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}
	ns, _, err := jxenv.GetDevNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to find dev namespace in %s", o.Namespace)
	}
	if ns != "" {
		o.Namespace = ns
	}
	if o.AllWebhooks == false && o.Filter == "" {
		return errors.New("set either --filter or --all-webhooks to delete webhooks")
	}

	return nil
}

// Run runs the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	jxClient := o.JXClient
	ns := o.Namespace

	srList, err := jxClient.JenkinsV1().SourceRepositories(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to find any SourceRepositories in namespace %s", ns)
	}

	for _, sr := range srList.Items {
		sourceRepo := sr

		_, err2 := o.DeleteWebhookFromSourceRepository(&sourceRepo, err, o.Filter)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

// DeleteWebhookFromSourceRepository updates the webhook for the given source repository
func (o *Options) DeleteWebhookFromSourceRepository(sr *v1.SourceRepository, err error, filter string) (bool, error) {
	if !o.matchesRepository(sr) {
		return false, nil
	}
	err = o.deleteWebhookIfItExists(sr, filter)
	if err != nil {
		if !o.WarnOnFail {
			return false, err
		}
		log.Logger().Warnf(err.Error())
	}
	return true, nil
}

func (o *Options) deleteWebhookIfItExists(repository *v1.SourceRepository, filter string) error {
	spec := repository.Spec
	gitServerURL := spec.Provider
	owner := spec.Org
	repo := spec.Repo

	o.ScmClientFactory.GitServerURL = gitServerURL
	o.ScmClientFactory.GitKind = spec.ProviderKind

	scmClient, err := o.ScmClientFactory.Create()
	if err != nil {
		return errors.Wrapf(err, "failed to create Scm client for %s", spec.URL)
	}

	err = o.removeRepositoryWebhook(scmClient, owner, repo, filter)
	if err != nil {
		return errors.Wrapf(err, "failed to update webhooks for Owner: %s and Repository: %s in git server: %s", owner, repo, gitServerURL)
	}

	return err
}

func (o *Options) removeRepositoryWebhook(scmClient *scm.Client, owner string, repoName string, filter string) error {
	fullName := scm.Join(owner, repoName)

	log.Logger().Debugf("Checking hooks for repository %s", info(fullName))

	ctx := context.Background()
	hooks, _, err := scmClient.Repositories.ListHooks(ctx, fullName, scm.ListOptions{})
	if err != nil {
		if !scmhelpers.IsScmNotFound(err) {
			log.Logger().Warnf("failed to find hooks for repository %s: %s", info(fullName), err.Error())
		}
	}

	// now lets remove any old ones
	if len(hooks) > 0 {
		// lets remove any previous matching hooks
		for _, hook := range hooks {
			if o.matchesFilter(hook, filter) {
				// lets remove any old ones
				log.Logger().Infof("repository %s has hook for url %s, removing it", info(fullName), info(hook.Target))
				if !o.DryRun {
					_, err = scmClient.Repositories.DeleteHook(ctx, fullName, hook.ID)
				} else {
					log.Logger().Infof("not deleting as dry-run is enabled")
				}
				if err != nil {
					return errors.Wrapf(err, "failed to delete webhook %s with target %s", hook.ID, hook.Target)
				}
			}
		}
	}

	return nil
}

func (o *Options) matchesFilter(webHookArgs *scm.Hook, filter string) bool {
	if o.AllWebhooks == false {
		return strings.Contains(webHookArgs.Target, filter)
	}
	return true
}

// matchesRepository returns true if the given source repository matchesFilter the current filters
func (o *Options) matchesRepository(repository *v1.SourceRepository) bool {
	if o.Org != "" && o.Org != repository.Spec.Org {
		return false
	}
	if o.Repo != "" && o.Repo != repository.Spec.Repo {
		return false
	}
	return true
}
