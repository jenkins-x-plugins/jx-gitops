package activities

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	jv1 "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	lhclient "github.com/jenkins-x/lighthouse-client/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Options command line arguments and flags
type Options struct {
	DryRun                  bool
	ReleaseHistoryLimit     int
	PullRequestHistoryLimit int
	ReleaseAgeLimit         time.Duration
	PullRequestAgeLimit     time.Duration
	PipelineRunAgeLimit     time.Duration
	ProwJobAgeLimit         time.Duration
	Namespace               string
	JXClient                jxc.Interface
	LHClient                lhclient.Interface
	DynamicClient           dynamic.Interface
}

const PrLabel = "tekton.dev/pipeline"

var (
	PipelineResource = schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1",
		Resource: "PipelineRun",
	}

	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Garbage collect the Jenkins X PipelineActivity resources

`)

	cmdExample = templates.Examples(`
		# garbage collect PipelineActivity resources
		jx gitops gc activities

		# dry run mode
		jx gitops gc pa --dry-run
`)
)

type buildCounter struct {
	ReleaseCount int
	PRCount      int
}

type buildsCount struct {
	cache map[string]*buildCounter
}

// AddBuild adds the build and returns the number of builds for this repo and branch
func (c *buildsCount) AddBuild(repoAndBranch string, isPR bool) int {
	if c.cache == nil {
		c.cache = map[string]*buildCounter{}
	}
	bc := c.cache[repoAndBranch]
	if bc == nil {
		bc = &buildCounter{}
		c.cache[repoAndBranch] = bc
	}
	if isPR {
		bc.PRCount++
		return bc.PRCount
	}
	bc.ReleaseCount++
	return bc.ReleaseCount
}

// NewCmd s a command object for the "step" command
func NewCmdGCActivities() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "activities",
		Aliases: []string{"pa", "act", "pr"},
		Short:   "garbage collection for PipelineActivity resources",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "d", false, "Dry run mode. If enabled just list the resources that would be removed")
	cmd.Flags().IntVarP(&o.ReleaseHistoryLimit, "release-history-limit", "l", 5, "Maximum number of PipelineActivities to keep around per repository release")
	cmd.Flags().IntVarP(&o.PullRequestHistoryLimit, "pr-history-limit", "", 2, "Minimum number of PipelineActivities to keep around per repository Pull Request")
	cmd.Flags().DurationVarP(&o.PullRequestAgeLimit, "pull-request-age", "p", time.Hour*48, "Maximum age to keep PipelineActivities for Pull Requests")
	cmd.Flags().DurationVarP(&o.ReleaseAgeLimit, "release-age", "r", time.Hour*24*30, "Maximum age to keep PipelineActivities for Releases")
	cmd.Flags().DurationVarP(&o.PipelineRunAgeLimit, "pipelinerun-age", "", time.Hour*12, "Maximum age to keep completed PipelineRuns for all pipelines")
	cmd.Flags().DurationVarP(&o.ProwJobAgeLimit, "prowjob-age", "", time.Hour*24*7, "Maximum age to keep completed ProwJobs for all pipelines")
	return cmd, o
}

// Run implements this command
func (o *Options) Run() error {
	var err error
	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}
	o.LHClient, err = LazyCreateLHClient(o.LHClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create the lighthouse client")
	}

	o.DynamicClient, err = kube.LazyCreateDynamicClient(o.DynamicClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create the tekton client")
	}

	client := o.JXClient
	currentNs := o.Namespace
	ctx := context.TODO()

	// cannot use field selectors like `spec.kind=Preview` on CRDs so list all environments
	activityInterface := client.JenkinsV1().PipelineActivities(currentNs)
	activities, err := activityInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(activities.Items) == 0 {
		// no preview environments found so lets return gracefully
		log.Logger().Debug("no activities found")
		return nil
	}

	now := time.Now()
	counters := &buildsCount{}

	var completedActivities []v1.PipelineActivity

	// Filter out running activities
	for k := range activities.Items {
		a := activities.Items[k]
		// TODO: Should we let activities with status pending lay around forever?
		if a.Spec.Status.IsTerminated() {
			completedActivities = append(completedActivities, a)
		}
	}

	// Sort with newest created activities first
	sort.Slice(completedActivities, func(i, j int) bool {
		return !completedActivities[i].Spec.CompletedTimestamp.Before(completedActivities[j].Spec.CompletedTimestamp)
	})

	//
	for k := range completedActivities {
		activity := completedActivities[k]
		branchName := activity.BranchName()
		isPR, isBatch := o.isPullRequestOrBatchBranch(branchName)
		maxAge, revisionHistory := o.ageAndHistoryLimits(isPR, isBatch)
		// lets remove activities that are too old
		timestamp := activity.Spec.CompletedTimestamp
		if timestamp == nil {
			timestamp = activity.Spec.StartedTimestamp
		}

		if timestamp.Add(maxAge).Before(now) {
			err = o.deleteResources(ctx, activityInterface, &activity, currentNs)
			if err != nil {
				return err
			}
			continue
		}

		repoBranchAndContext := activity.RepositoryOwner() + "/" + activity.RepositoryName() + "/" + activity.BranchName() + "/" + activity.Spec.Context
		c := counters.AddBuild(repoBranchAndContext, isPR)
		if c > revisionHistory && timestamp != nil {
			err = o.deleteResources(ctx, activityInterface, &activity, currentNs)
			if err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

func (o *Options) deleteResources(ctx context.Context, activityInterface jv1.PipelineActivityInterface, a *v1.PipelineActivity, currentNs string) error {
	err := o.deleteLighthouseJob(ctx, a)
	if err != nil {
		return err
	}

	prName := a.Labels[PrLabel]
	pr, err := o.getPipelineRun(ctx, currentNs, prName)
	if err != nil {
		log.Logger().Warnf("pipelinerun %s not found, skipping", prName)
	}

	// Delete only existing pipelineRuns, need to check for error, as we dont return err in the step before
	if pr != nil && err == nil {
		err = o.deletePipelineRun(ctx, currentNs, prName)
		if err != nil {
			log.Logger().Warn(err.Error())
		}
	}
	err = o.deleteActivity(ctx, activityInterface, a)
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) deleteActivity(ctx context.Context, activityInterface jv1.PipelineActivityInterface, a *v1.PipelineActivity) error {
	prefix := ""
	if o.DryRun {
		prefix = "not "
	}
	log.Logger().Infof("%sdeleting PipelineActivity %s", prefix, info(a.Name))
	if o.DryRun {
		return nil
	}
	return activityInterface.Delete(ctx, a.Name, *metav1.NewDeleteOptions(0))
}

func (o *Options) ageAndHistoryLimits(isPR, isBatch bool) (time.Duration, int) {
	maxAge := o.ReleaseAgeLimit
	revisionLimit := o.ReleaseHistoryLimit
	if isPR || isBatch {
		maxAge = o.PullRequestAgeLimit
		revisionLimit = o.PullRequestHistoryLimit
	}
	return maxAge, revisionLimit
}

func (o *Options) isPullRequestOrBatchBranch(branchName string) (bool, bool) {
	return strings.HasPrefix(branchName, "PR-"), branchName == "batch"
}

func (o *Options) deleteLighthouseJob(ctx context.Context, pa *v1.PipelineActivity) error {
	if pa.Labels == nil {
		return nil
	}
	labelMap := map[string]string{}
	for k, v := range pa.Labels {
		if k != "lighthouse.jenkins-x.io/id" && strings.HasPrefix(k, "lighthouse.jenkins-x.io/") {
			labelMap[k] = v
		}
	}
	if len(labelMap) < 4 {
		log.Logger().Infof("ignoring PipelineActivity %s which only has lighthouse labels %v", pa.Name, labelMap)
		return nil
	}
	selector := labels.SelectorFromSet(labelMap).String()
	lighthouseJobInterface := o.LHClient.LighthouseV1alpha1().LighthouseJobs(o.Namespace)
	list, err := lighthouseJobInterface.List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to list LighthouseJob resources with selector %s", selector)
	}
	if list == nil {
		return nil
	}
	for i := range list.Items {
		r := &list.Items[i]
		prefix := ""
		if o.DryRun {
			prefix = "not "
		}
		log.Logger().Infof("%sdeleting LighthouseJob %s", prefix, info(r.Name))
		if o.DryRun {
			continue
		}

		err = lighthouseJobInterface.Delete(ctx, r.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete LighthouseJob %s", r.Name)
		}
	}
	return nil
}

func (o *Options) deletePipelineRun(ctx context.Context, ns, prName string) error {
	prefix := ""
	if o.DryRun {
		prefix = "not "
	}
	log.Logger().Infof("%sdeleting PipelineRun %s", prefix, info(prName))
	if o.DryRun {
		return nil
	}
	return o.DynamicClient.Resource(PipelineResource).Namespace(ns).Delete(ctx, prName, metav1.DeleteOptions{})
}

func (o *Options) getPipelineRun(ctx context.Context, ns, prName string) (*unstructured.Unstructured, error) {
	return o.DynamicClient.Resource(PipelineResource).Namespace(ns).Get(ctx, prName, metav1.GetOptions{})
}

// LazyCreateLHClient lazy creates the lighthouse client if its not defined
func LazyCreateLHClient(client lhclient.Interface) (lhclient.Interface, error) {
	if client != nil {
		return client, nil
	}
	f := kubeclient.NewFactory()
	cfg, err := f.CreateKubeConfig()
	if err != nil {
		return client, errors.Wrap(err, "failed to get kubernetes config")
	}
	client, err = lhclient.NewForConfig(cfg)
	if err != nil {
		return client, errors.Wrap(err, "error building lighthouse clientset")
	}
	return client, nil
}
