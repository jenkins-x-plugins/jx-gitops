package pods

import (
	"context"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"strings"
	"time"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/errorutil"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Options containers the CLI options
type Options struct {
	Selector   string
	Namespace  string
	Age        time.Duration
	KubeClient kubernetes.Interface
}

const (
	JXEnvironmentLabel = "env"
	JXGitOpsLabel      = "gitops.jenkins-x.io/pipeline"
)

var (
	cmdLong = templates.LongDesc(`
		Garbage collect old Pods that have completed or failed
`)

	cmdExample = templates.Examples(`
		# garbage collect old pods of the default age
		jx gitops gc pods

		# garbage collect pods older than 10 minutes
		jx gitops gc pods -a 10m

`)
)

// NewCmdGCPods creates the command object
func NewCmdGCPods() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "pods",
		Short:   "garbage collection for pods",
		Aliases: []string{"pod"},
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Selector, "selector", "s", "", "The selector to use to filter the pods")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "The namespace to look for the pods. If empty will work in namespaces labelled by 'env' and 'gitops.jenkins-x.io/pipeline', also will include `'jx-git-operator' namespace")
	cmd.Flags().DurationVarP(&o.Age, "age", "a", time.Hour, "The minimum age of pods to garbage collect. Any newer pods will be kept")
	return cmd, o
}

// Run implements this command
func (o *Options) Run() error {
	var err error
	o.KubeClient, _, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}

	kubeClient := o.KubeClient
	ctx := context.TODO()

	var namespaces []string
	if o.Namespace != "" {
		log.Logger().Infof("Using a fixed namespace '%s'", o.Namespace)

		// run in single, selected namespace
		namespaces = []string{o.Namespace}
	} else {
		log.Logger().Info("Looking for JX namespaces")

		// run in all namespaces that are owned by Jenkins X
		namespaces, err = o.findJXNamespaces(ctx)
		if err != nil {
			return errors.Wrap(err, "cannot perform garbage collection of pods")
		}
	}

	selector := o.Selector
	var collectedErrors []error

	for _, ns := range namespaces {
		podInterface := kubeClient.CoreV1().Pods(ns)
		collectedErrors = append(collectedErrors, o.runInNamespace(ctx, ns, podInterface, selector)...)
	}

	return errorutil.CombineErrors(collectedErrors...)
}

// runInNamespace runs a garbage collection in context of a given namespace
func (o *Options) runInNamespace(ctx context.Context, namespace string, podInterface v1.PodInterface, selector string) []error {
	opts := metav1.ListOptions{
		LabelSelector: selector,
	}

	podList, err := podInterface.List(ctx, opts)
	if err != nil {
		return []error{err}
	}

	deleteOptions := metav1.DeleteOptions{}
	var collectedErrors []error

	for k := range podList.Items {
		pod := podList.Items[k]
		matches, age := o.MatchesPod(&pod)
		if matches {
			err := podInterface.Delete(ctx, pod.Name, deleteOptions)
			if err != nil {
				log.Logger().Warnf("Failed to delete pod %s in namespace %s: %s", pod.Name, namespace, err)
				collectedErrors = append(collectedErrors, err)
			} else {
				ageText := strings.TrimSuffix(age.Round(time.Minute).String(), "0s")
				log.Logger().Infof("Deleted pod %s in namespace %s with phase %s as its age is: %s", pod.Name, namespace, string(pod.Status.Phase), ageText)
			}
		}
	}
	return collectedErrors
}

// MatchesPod returns true if this pod can be garbage collected
func (o *Options) MatchesPod(pod *corev1.Pod) (bool, time.Duration) {
	phase := pod.Status.Phase
	now := time.Now()

	finished := now.Add(-1000 * time.Hour)
	for k := range pod.Status.ContainerStatuses {
		terminated := pod.Status.ContainerStatuses[k].State.Terminated
		if terminated != nil {
			if terminated.FinishedAt.After(finished) {
				finished = terminated.FinishedAt.Time
			}
		}
	}
	age := now.Sub(finished)
	if phase != corev1.PodSucceeded && phase != corev1.PodFailed {
		return false, age
	}
	return age > o.Age, age
}

// findJXNamespaces looks for namespace names that are owned by Jenkins X
func (o *Options) findJXNamespaces(ctx context.Context) ([]string, error) {
	var matched []string
	namespaces, err := o.KubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []string{}, errors.Wrap(err, "cannot list namespaces to look for pods")
	}
	for _, n := range namespaces.Items {
		labels := n.GetLabels()

		if _, okEnv := labels[JXEnvironmentLabel]; okEnv {
			if _, okGitOps := labels[JXGitOpsLabel]; okGitOps {
				log.Logger().Infof("Found Jenkins X namespace '%s' by environment labels", n.Name)
				matched = append(matched, n.Name)
			}
		}

		if n.Name == "jx-git-operator" {
			log.Logger().Infof("Found Jenkins X git operator namespace '%s'", n.Name)
			matched = append(matched, n.Name)
		}
	}
	return matched, nil
}
