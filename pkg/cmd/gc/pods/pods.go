package pods

import (
	"context"
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
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "The namespace to look for the pods. Defaults to the current namespace")
	cmd.Flags().DurationVarP(&o.Age, "age", "a", time.Hour, "The minimum age of pods to garbage collect. Any newer pods will be kept")
	return cmd, o
}

// Run implements this command
func (o *Options) Run() error {
	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}

	kubeClient := o.KubeClient
	ns := o.Namespace
	ctx := context.TODO()

	opts := metav1.ListOptions{
		LabelSelector: o.Selector,
	}
	podInterface := kubeClient.CoreV1().Pods(ns)
	podList, err := podInterface.List(ctx, opts)
	if err != nil {
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	errors := []error{}
	for k := range podList.Items {
		pod := podList.Items[k]
		matches, age := o.MatchesPod(&pod)
		if matches {
			err := podInterface.Delete(ctx, pod.Name, deleteOptions)
			if err != nil {
				log.Logger().Warnf("Failed to delete pod %s in namespace %s: %s", pod.Name, ns, err)
				errors = append(errors, err)
			} else {
				ageText := strings.TrimSuffix(age.Round(time.Minute).String(), "0s")
				log.Logger().Infof("Deleted pod %s in namespace %s with phase %s as its age is: %s", pod.Name, ns, string(pod.Status.Phase), ageText)
			}
		}
	}
	return errorutil.CombineErrors(errors...)
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
