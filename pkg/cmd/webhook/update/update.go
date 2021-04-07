package update

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/go-scm/scm"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/services"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
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
	ExactHookMatch   bool
	PreviousHookUrl  string
	HMAC             string
	Endpoint         string
	DryRun           bool
	WarnOnFail       bool
	Namespace        string
	KubeClient       kubernetes.Interface
	JXClient         jxc.Interface
}

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Updates the webhooks for all the source repositories optionally filtering by owner and/or repository

`)

	cmdExample = templates.Examples(`
		# update all the webhooks for all SourceRepository and Environment resource:
		%s update

		# only update the webhooks for a given owner
		%s update --org=mycorp

		# use a custom hook webhook endpoint (e.g. if you are on premise using node ports or something)
		%s update --endpoint http://mything.com

`)
)

func NewCmdWebHookVerify() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Updates the webhooks for all the source repositories optionally filtering by owner and/or repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName, rootcmd.BinaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Org, "owner", "o", "", "The name of the git organisation or user to filter on")
	cmd.Flags().StringVarP(&o.Repo, "repo", "r", "", "The name of the repository to filter on")
	cmd.Flags().BoolVarP(&o.ExactHookMatch, "exact-hook-url-match", "", true, "Whether to exactly match the hook based on the URL")
	cmd.Flags().StringVarP(&o.PreviousHookUrl, "previous-hook-url", "", "", "Whether to match based on an another URL")
	cmd.Flags().StringVarP(&o.HMAC, "hmac", "", "", "Don't use the HMAC token from the cluster, use the provided token")
	cmd.Flags().StringVarP(&o.Endpoint, "endpoint", "", "", "Don't use the endpoint from the cluster, use the provided endpoint")
	cmd.Flags().BoolVarP(&o.WarnOnFail, "warn-on-fail", "", false, "If enabled lets just log a warning that we could not update the webhook")

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

	if o.Endpoint == "" {
		o.Endpoint, err = o.GetWebHookEndpointFromHook()
		if err != nil {
			return errors.Wrapf(err, "failed to find webhook endpoint")
		}
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
	envMap, _, err := jxenv.GetEnvironments(jxClient, ns)

	for _, sr := range srList.Items {
		sourceRepo := sr

		// hmac isn't supported on bitbucketcloud
		if sr.Spec.ProviderKind != "bitbucketcloud" && o.HMAC == "" {
			o.HMAC, err = o.GetHMACTokenFromSecret()
			if err != nil {
				return errors.Wrapf(err, "failed to find hmac token from secret")
			}
		}

		_, err2 := o.UpdateWebhookForSourceRepository(&sourceRepo, envMap, err, o.Endpoint, o.HMAC)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

// GetWebHookEndpointFromHook returns the webhook endpoint
func (o *Options) GetWebHookEndpointFromHook() (string, error) {
	baseURL, err := services.GetServiceURLFromName(o.KubeClient, "hook", o.Namespace)
	if err != nil {
		return "", err
	}

	// lets add /hook if it does not already have it
	if !strings.HasSuffix(baseURL, "/hook") {
		baseURL = stringhelpers.UrlJoin(baseURL, "hook")
	}
	return baseURL, nil
}

// GetHMACTokenFromSecret gets the appropriate HMAC secret, for either Prow or Lighthouse
func (o *Options) GetHMACTokenFromSecret() (string, error) {
	kubeClient := o.KubeClient
	ns := o.Namespace
	name := LighthouseHMACToken
	hmacTokenSecret, err := kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "could not find lighthouse hmac token %s in namespace %s", name, ns)
	}
	hmac := string(hmacTokenSecret.Data["hmac"])
	if hmac == "" {
		return hmac, errors.Errorf("secret %s in namespace %s has no key: hmac", name, ns)
	}
	return hmac, nil
}

// UpdateWebhookForSourceRepository updates the webhook for the given source repository
func (o *Options) UpdateWebhookForSourceRepository(sr *v1.SourceRepository, envMap map[string]*v1.Environment, err error, webhookURL string, hmacToken string) (bool, error) {
	if !o.matchesRepository(sr) {
		return false, nil
	}
	err = o.ensureWebHookCreated(sr, webhookURL, hmacToken)
	if err != nil {
		if !o.WarnOnFail {
			return false, err
		}
		log.Logger().Warnf(err.Error())
	}
	return true, nil
}

func (o *Options) ensureWebHookCreated(repository *v1.SourceRepository, webhookURL string, hmacToken string) error {
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

	srInterface := o.JXClient.JenkinsV1().SourceRepositories(o.Namespace)
	if repository.Annotations == nil {
		repository.Annotations = map[string]string{}
	}
	annotate := func() {
		var err2 error
		repository, err2 = srInterface.Update(context.TODO(), repository, metav1.UpdateOptions{})
		if err2 != nil {
			log.Logger().Warnf("failed to annotate SourceRepository %s with webhook status: %s", repository.Name, err2.Error())
		}
		if repository.Annotations == nil {
			repository.Annotations = map[string]string{}
		}
	}

	repository.Annotations[WebHookAnnotation] = "creating"
	annotate()

	err = o.updateRepositoryWebhook(scmClient, owner, repo, webhookURL, hmacToken)
	if err != nil {
		repository.Annotations[WebHookAnnotation] = "failed"
		repository.Annotations[WebHookErrorAnnotation] = err.Error()
		annotate()
		return errors.Wrapf(err, "failed to update webhooks for Owner: %s and Repository: %s in git server: %s", owner, repo, gitServerURL)
	}

	repository.Annotations[WebHookAnnotation] = "true"
	annotate()
	return err
}

func (o *Options) updateRepositoryWebhook(scmClient *scm.Client, owner string, repoName string, webhookURL string, hmacToken string) error {
	fullName := scm.Join(owner, repoName)

	log.Logger().Debugf("Checking hooks for repository %s", info(fullName))

	ctx := context.Background()
	hooks, _, err := scmClient.Repositories.ListHooks(ctx, fullName, scm.ListOptions{})
	if err != nil {
		if !scmhelpers.IsScmNotFound(err) {
			log.Logger().Warnf("failed to find hooks for repository %s: %s", info(fullName), err.Error())
		}
	}

	skipVerify := false
	requirements, _, err := jxcore.LoadRequirementsConfig("", false)
	if err != nil {
		log.Logger().Warnf("unable to load requirements from the local directory so defaulting skipVerify option on the webhook to false")
	}
	if requirements != nil {
		if requirements.Spec.Ingress.TLS != nil {
			skipVerify = !requirements.Spec.Ingress.TLS.Production
		}
	}

	webHookArgs := &scm.HookInput{
		Name:   "",
		Target: webhookURL,
		Secret: hmacToken,
		Events: scm.HookEvents{
			Branch:             true,
			Deployment:         true,
			DeploymentStatus:   true,
			Issue:              true,
			IssueComment:       true,
			PullRequest:        true,
			PullRequestComment: true,
			Push:               true,
			Review:             true,
			ReviewComment:      true,
			Tag:                true,
		},
		SkipVerify:   skipVerify,
		NativeEvents: nil,
	}

	// now lets remove any old ones
	if len(hooks) > 0 {
		// lets remove any previous matching hooks
		for _, hook := range hooks {
			if o.matchesWebhookURL(hook, webhookURL) {
				// lets remove any old ones
				log.Logger().Infof("repository %s has hook for url %s", info(fullName), info(hook.Target))
				_, err = scmClient.Repositories.DeleteHook(ctx, fullName, hook.ID)
				if err != nil {
					return errors.Wrapf(err, "failed to delete webhook %s with target %s", hook.ID, hook.Target)
				}
			}
		}
	}

	// lets create a new webhook...
	_, resp, err := scmClient.Repositories.CreateHook(ctx, fullName, webHookArgs)
	if err != nil {
		status := ""
		if resp != nil && resp.Body != nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil && body != nil {
				status = " " + string(body)
			}
		}
		return errors.Wrapf(err, "failed to create webhook %q on repository '%s'%s", webhookURL, fullName, status)
	}
	return nil
}

func (o *Options) matchesWebhookURL(webHookArgs *scm.Hook, webhookURL string) bool {
	if "" != o.PreviousHookUrl {
		return o.PreviousHookUrl == webHookArgs.Target
	}
	if o.ScmClientFactory.GitKind == "gitlab" || o.ScmClientFactory.GitKind == "gitea" {
		return strings.HasPrefix(webHookArgs.Target, webhookURL)
	}
	if o.ExactHookMatch {
		return webhookURL == webHookArgs.Target
	} else {
		return strings.Contains(webHookArgs.Target, "hook")
	}
}

// matchesRepository returns true if the given source repository matchesWebhookURL the current filters
func (o *Options) matchesRepository(repository *v1.SourceRepository) bool {
	if o.Org != "" && o.Org != repository.Spec.Org {
		return false
	}
	if o.Repo != "" && o.Repo != repository.Spec.Repo {
		return false
	}
	return true
}
