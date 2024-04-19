package jobs

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/errorutil"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Options containers the CLI options
type Options struct {
	DryRun     bool
	Selector   string
	Namespace  string
	Age        time.Duration
	Keep       int
	KubeClient kubernetes.Interface
}

var (
	cmdLong = templates.LongDesc(`
		Garbage collect old Jobs that have completed or failed
`)

	cmdExample = templates.Examples(`
		# garbage collect old jobs of the default age keeping 1
		jx gitops gc jobs

		# garbage collect jobs older than 10 minutes and keeping 10
		jx gitops gc jobs -a 10m -k 10
		
		# garbage collect jobs older than 10 (don't keep any job)
		jx gitops gc jobs -a 10m -k 0
		
		# dry run mode
		jx gitops gc jobs --dry-run

`)
)

// NewCmdGCJods creates the command object
func NewCmdGCJobs() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "jobs",
		Short:   "garbage collection for jobs",
		Aliases: []string{"job"},
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "d", false, "Dry run mode. If enabled just list the jobs that would be removed")
	cmd.Flags().StringVarP(&o.Selector, "selector", "s", "", "The selector to use to filter the jobs")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "The namespace to look for the jobs. Defaults to the current namespace")
	cmd.Flags().DurationVarP(&o.Age, "age", "a", time.Hour, "The minimum age of jobs to garbage collect. Any newer jobs will be kept")
	cmd.Flags().IntVarP(&o.Keep, "keep", "k", 1, "The minimum jobs to keep. Jobs to keep even if they are older than the age parameter")
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
	jobInterface := kubeClient.BatchV1().Jobs(o.Namespace)
	jobList, err := jobInterface.List(ctx, opts)
	if err != nil {
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	errors := []error{}

	// we need to keep all jobs, don't waste time sorting or iterating
	if o.Keep >= len(jobList.Items) {
		log.Logger().Infof("Not deleting any job. You want to keep %d jobs and we have %d jobs in the list.", o.Keep, len(jobList.Items))
		return errorutil.CombineErrors(errors...)
	}

	// if you don't want keep any job don't waste time sorting
	if o.Keep != 0 {
		// sort list by creationTimestamp
		sort.Slice(jobList.Items, func(i, j int) bool {
			return jobList.Items[j].Status.StartTime.Before(jobList.Items[i].Status.StartTime)
		})

		// shrink jobList by keep
		jobList.Items = jobList.Items[o.Keep:]
	}

	for k := range jobList.Items {
		job := jobList.Items[k]
		matches, age := o.matchesJob(&job)
		ageText := strings.TrimSuffix(age.Round(time.Minute).String(), "0s")
		if matches {
			if !o.DryRun {
				err := jobInterface.Delete(ctx, job.Name, deleteOptions)
				if err != nil {
					log.Logger().Warnf("Failed to delete job %s in namespace %s: %s", job.Name, ns, err.Error())
					errors = append(errors, err)
				} else {
					log.Logger().Infof("Deleted job %s in namespace %s as its age is: %s", job.Name, ns, ageText)
				}
			} else {
				log.Logger().Infof("Not deleting job %s in namespace %s. It's age is: %s", job.Name, ns, ageText)
			}
		}
	}
	return errorutil.CombineErrors(errors...)
}

// matchesJob returns true if this job can be garbage collected
func (o *Options) matchesJob(job *batchv1.Job) (bool, time.Duration) {
	now := time.Now()
	var age time.Duration
	if job.Status.StartTime != nil {
		age = now.Sub(job.Status.StartTime.Time)
	} else {
		return false, age
	}
	if job.Status.Active > 0 {
		return false, age
	}
	return age > o.Age, age
}
